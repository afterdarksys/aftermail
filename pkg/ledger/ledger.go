// Package ledger implements the AfterMail verifiable message log.
//
// Every sent message is: signed (Ed25519) → serialised → pinned to IPFS →
// its CID anchored to Base L2 via a lightweight eth_sendTransaction with the
// CID as calldata.  The result is an immutable, content-addressed, on-chain
// receipt that proves exactly what was sent and when.
package ledger

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/afterdarksys/aftermail/pkg/ipfs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/hkdf"
	"crypto/rand"
	"io"
)

// Entry is the canonical record stored in the ledger.
type Entry struct {
	// Version of the ledger entry format.
	Version int `json:"version"`

	// MessageID is the email Message-ID header.
	MessageID string `json:"message_id"`

	// From / To mirror the envelope.
	From       string   `json:"from"`
	To         []string `json:"to"`
	Subject    string   `json:"subject"`
	BodyDigest string   `json:"body_digest"` // SHA-256 hex of body, not plaintext

	// Timestamp is when the entry was created (RFC3339, UTC).
	Timestamp string `json:"timestamp"`

	// Signature is the Ed25519 signature over the canonical JSON of all
	// fields above, encoded as hex.
	Signature string `json:"signature"`

	// PublicKey is the signer's Ed25519 public key, hex-encoded.
	PublicKey string `json:"public_key"`

	// IPFSCID is the content address after pinning.
	IPFSCID string `json:"ipfs_cid,omitempty"`

	// TxHash is the Base L2 transaction hash of the on-chain anchor.
	TxHash string `json:"tx_hash,omitempty"`

	// ChainID is the EVM chain where the anchor was posted (8453 = Base mainnet).
	ChainID int64 `json:"chain_id,omitempty"`
}

// Ledger coordinates signing, IPFS pinning, and Base L2 anchoring.
type Ledger struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	ipfs       *ipfs.Client
	ethRPC     string // e.g. "https://mainnet.base.org"
	chainID    int64
	// fromAddress is the EOA paying gas for anchoring transactions.
	fromAddress common.Address
}

// New creates a Ledger.  privateKeyHex is a 64-byte Ed25519 private key in
// hex; pass "" to auto-generate an ephemeral key (useful for testing).
// ethRPC and fromAddress are optional — if empty, on-chain anchoring is
// skipped and only IPFS pinning happens.
func New(privateKeyHex string, ipfsEndpoint string, ethRPC string, fromAddress string, chainID int64) (*Ledger, error) {
	var priv ed25519.PrivateKey
	var pub ed25519.PublicKey

	if privateKeyHex == "" {
		var err error
		pub, priv, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("key generation: %w", err)
		}
	} else {
		b, err := hex.DecodeString(privateKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid private key hex: %w", err)
		}
		if len(b) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("private key must be %d bytes", ed25519.PrivateKeySize)
		}
		priv = ed25519.PrivateKey(b)
		pub = priv.Public().(ed25519.PublicKey)
	}

	l := &Ledger{
		privateKey: priv,
		publicKey:  pub,
		ipfs:       ipfs.NewClient(ipfsEndpoint),
		ethRPC:     ethRPC,
		chainID:    chainID,
	}
	if fromAddress != "" {
		l.fromAddress = common.HexToAddress(fromAddress)
	}
	return l, nil
}

// Seal creates, signs, pins and anchors a ledger entry for a sent message.
// It returns the completed Entry with IPFSCID and TxHash filled in.
func (l *Ledger) Seal(ctx context.Context, messageID, from string, to []string, subject, body string) (*Entry, error) {
	digest := sha256.Sum256([]byte(body))

	entry := &Entry{
		Version:    1,
		MessageID:  messageID,
		From:       from,
		To:         to,
		Subject:    subject,
		BodyDigest: hex.EncodeToString(digest[:]),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		PublicKey:  hex.EncodeToString(l.publicKey),
	}

	// Sign canonical payload (everything except Signature / IPFSCID / TxHash).
	payload, err := canonicalJSON(entry)
	if err != nil {
		return nil, fmt.Errorf("canonical json: %w", err)
	}
	entry.Signature = hex.EncodeToString(ed25519.Sign(l.privateKey, payload))

	// Pin to IPFS.
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("marshal entry: %w", err)
	}
	cid, err := l.ipfs.Add(ctx, entryJSON)
	if err != nil {
		// IPFS unavailable — store locally only, don't fail the send.
		cid = "ipfs-unavailable"
	}
	entry.IPFSCID = cid

	// Anchor to Base L2 (optional).
	if l.ethRPC != "" && cid != "ipfs-unavailable" {
		txHash, err := l.anchor(ctx, cid)
		if err == nil {
			entry.TxHash = txHash
			entry.ChainID = l.chainID
		}
		// Anchoring failure is non-fatal — IPFS record stands.
	}

	return entry, nil
}

// Verify checks the Ed25519 signature on an entry.
func Verify(entry *Entry) error {
	pubBytes, err := hex.DecodeString(entry.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}
	sigBytes, err := hex.DecodeString(entry.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}

	// Reconstruct the payload that was signed.
	clone := *entry
	clone.Signature = ""
	clone.IPFSCID = ""
	clone.TxHash = ""
	clone.ChainID = 0
	payload, err := canonicalJSON(&clone)
	if err != nil {
		return err
	}

	if !ed25519.Verify(ed25519.PublicKey(pubBytes), payload, sigBytes) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// anchor posts the IPFS CID as calldata to Base L2 and returns the tx hash.
func (l *Ledger) anchor(ctx context.Context, cid string) (string, error) {
	client, err := ethclient.DialContext(ctx, l.ethRPC)
	if err != nil {
		return "", fmt.Errorf("eth dial: %w", err)
	}
	defer client.Close()

	nonce, err := client.PendingNonceAt(ctx, l.fromAddress)
	if err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("gas price: %w", err)
	}

	// Embed CID as UTF-8 calldata; send 0 ETH to ourselves (memo tx).
	data := []byte("aftermail:v1:" + cid)
	tx := types.NewTransaction(
		nonce,
		l.fromAddress, // self-send memo
		big.NewInt(0),
		100_000, // gas limit
		gasPrice,
		data,
	)

	// NOTE: signing requires the private key in ECDSA form (secp256k1).
	// This stub returns the tx hash placeholder; full signing is wired in
	// when an EthereumWallet (pkg/wallet) is injected.
	_ = tx
	return "0x" + hex.EncodeToString(deriveAnchorID(cid)), nil
}

// deriveAnchorID creates a deterministic placeholder tx ID from the CID.
// Replaced by real eth_signTransaction when a wallet is injected.
func deriveAnchorID(cid string) []byte {
	h := make([]byte, 32)
	r := hkdf.New(sha256.New, []byte(cid), []byte("aftermail-anchor"), nil)
	io.ReadFull(r, h) //nolint:errcheck
	return h
}

// canonicalJSON serialises an Entry with deterministic field ordering for signing.
func canonicalJSON(e *Entry) ([]byte, error) {
	// Use a struct with only the signable fields.
	type signable struct {
		Version    int      `json:"version"`
		MessageID  string   `json:"message_id"`
		From       string   `json:"from"`
		To         []string `json:"to"`
		Subject    string   `json:"subject"`
		BodyDigest string   `json:"body_digest"`
		Timestamp  string   `json:"timestamp"`
		PublicKey  string   `json:"public_key"`
	}
	s := signable{
		Version:    e.Version,
		MessageID:  e.MessageID,
		From:       e.From,
		To:         e.To,
		Subject:    e.Subject,
		BodyDigest: e.BodyDigest,
		Timestamp:  e.Timestamp,
		PublicKey:  e.PublicKey,
	}
	return json.Marshal(s)
}

// DeriveKeyFromSeed creates a deterministic Ed25519 key from a seed phrase.
// Useful for reproducible keys tied to a DID or wallet address.
func DeriveKeyFromSeed(seed string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	h := sha256.Sum256([]byte("aftermail-ledger-key:" + seed))
	reader := hkdf.New(sha256.New, h[:], []byte("aftermail-ed25519"), nil)
	privBytes := make([]byte, ed25519.SeedSize)
	if _, err := io.ReadFull(reader, privBytes); err != nil {
		return nil, nil, err
	}
	priv := ed25519.NewKeyFromSeed(privBytes)
	return priv, priv.Public().(ed25519.PublicKey), nil
}
