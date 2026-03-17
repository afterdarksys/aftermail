package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EthereumWallet manages Ethereum wallet operations
type EthereumWallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	Address    common.Address
}

// NewWallet creates a new Ethereum wallet
func NewWallet() (*EthereumWallet, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &EthereumWallet{
		PrivateKey: privateKey,
		PublicKey:  publicKeyECDSA,
		Address:    address,
	}, nil
}

// FromPrivateKeyHex imports wallet from hex-encoded private key
func FromPrivateKeyHex(privateKeyHex string) (*EthereumWallet, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &EthereumWallet{
		PrivateKey: privateKey,
		PublicKey:  publicKeyECDSA,
		Address:    address,
	}, nil
}

// FromKeystore imports wallet from keystore file
func FromKeystore(keystorePath, password string) (*EthereumWallet, error) {
	ks := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
	if len(ks.Accounts()) == 0 {
		return nil, fmt.Errorf("no accounts found in keystore")
	}

	account := ks.Accounts()[0]
	keyJSON, err := ks.Export(account, password, password)
	if err != nil {
		return nil, fmt.Errorf("failed to export key: %w", err)
	}

	key, err := keystore.DecryptKey(keyJSON, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %w", err)
	}

	return &EthereumWallet{
		PrivateKey: key.PrivateKey,
		PublicKey:  &key.PrivateKey.PublicKey,
		Address:    key.Address,
	}, nil
}

// ExportPrivateKeyHex exports private key as hex string
func (w *EthereumWallet) ExportPrivateKeyHex() string {
	return hex.EncodeToString(crypto.FromECDSA(w.PrivateKey))
}

// ExportToKeystore exports wallet to keystore format
func (w *EthereumWallet) ExportToKeystore(keystorePath, password string) error {
	ks := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)

	_, err := ks.ImportECDSA(w.PrivateKey, password)
	if err != nil {
		return fmt.Errorf("failed to import to keystore: %w", err)
	}

	return nil
}

// Sign signs a message hash
func (w *EthereumWallet) Sign(hash []byte) ([]byte, error) {
	signature, err := crypto.Sign(hash, w.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}
	return signature, nil
}

// SignMessage signs a message with Ethereum prefix
func (w *EthereumWallet) SignMessage(message []byte) ([]byte, error) {
	hash := accounts.TextHash(message)
	return w.Sign(hash)
}

// VerifySignature verifies a signature
func VerifySignature(address common.Address, message []byte, signature []byte) bool {
	hash := accounts.TextHash(message)

	// Remove recovery ID
	if len(signature) == 65 {
		signature = signature[:64]
	}

	pubKey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return false
	}

	recoveredAddress := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddress == address
}

// FormatBalance formats wei to ether string
func FormatBalance(wei *big.Int) string {
	if wei == nil {
		return "0 ETH"
	}

	ether := new(big.Float).Quo(
		new(big.Float).SetInt(wei),
		big.NewFloat(1e18),
	)

	return fmt.Sprintf("%.6f ETH", ether)
}

// ParseEther converts ether string to wei
func ParseEther(ether string) (*big.Int, error) {
	eth, ok := new(big.Float).SetString(ether)
	if !ok {
		return nil, fmt.Errorf("invalid ether amount: %s", ether)
	}

	wei := new(big.Float).Mul(eth, big.NewFloat(1e18))
	result, _ := wei.Int(nil)

	return result, nil
}
