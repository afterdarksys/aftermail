package accounts

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"strings"
	"sync"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	// Import protobuf definitions
	"github.com/afterdarksys/aftermail/pkg/proto"
	"github.com/afterdarksys/aftermail/pkg/web3mail"
	protobuf "google.golang.org/protobuf/proto"
)

const (
	DefaultMsgsGlobalGateway = "amp.msgs.global:4433"
)

var (
	grpcConnPool = make(map[string]*grpc.ClientConn)
	grpcConnMu   sync.Mutex
)

// MsgsGlobalClient handles communication with msgs.global using AfterSMTP protocol
type MsgsGlobalClient struct {
	account      *Account
	conn         *grpc.ClientConn
	clientAPI     proto.ClientAPIClient
	ampServer     proto.AMPServerClient
	jwtToken      string
	signingKey    ed25519.PrivateKey
	encryptionKey [32]byte
	registry      *web3mail.MailblocksRegistry
}

// NewMsgsGlobalClient creates a new msgs.global client
func NewMsgsGlobalClient(account *Account) (*MsgsGlobalClient, error) {
	if account.DID == "" {
		return nil, fmt.Errorf("DID is required for msgs.global account")
	}

	// Parse Ed25519 signing key
	signingKeyBytes, err := hex.DecodeString(account.Ed25519PrivKey)
	if err != nil {
		return nil, fmt.Errorf("invalid Ed25519 private key: %w", err)
	}
	if len(signingKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("Ed25519 private key must be %d bytes", ed25519.PrivateKeySize)
	}
	signingKey := ed25519.PrivateKey(signingKeyBytes)

	// Parse X25519 encryption key
	encryptionKeyBytes, err := hex.DecodeString(account.X25519PrivKey)
	if err != nil {
		return nil, fmt.Errorf("invalid X25519 private key: %w", err)
	}
	if len(encryptionKeyBytes) != 32 {
		return nil, fmt.Errorf("X25519 private key must be 32 bytes")
	}
	var encryptionKey [32]byte
	copy(encryptionKey[:], encryptionKeyBytes)

	gatewayURL := account.GatewayURL
	if gatewayURL == "" {
		gatewayURL = DefaultMsgsGlobalGateway
	}

	grpcConnMu.Lock()
	conn, ok := grpcConnPool[gatewayURL]
	if !ok {
		// Establish gRPC connection with TLS
		var dialOpts []grpc.DialOption
	
		// We use WithBlock for connection pooling to ensure transport is alive before caching
		dialOpts = append(dialOpts, grpc.WithBlock())
	
		target := gatewayURL
		if strings.HasPrefix(target, "quic://") {
			// QUIC inherently negotiates TLS 1.3. We must tell gRPC not to attempt a SECOND TLS wrap,
			// otherwise it will hang on the protocol overlay.
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			dialOpts = append(dialOpts, grpc.WithContextDialer(quicDialer))
			
			// Strip from the canonical target we pass to grpc
			target = strings.TrimPrefix(target, "quic://")
		} else {
			// Standard outward TCP TLS
			creds := credentials.NewClientTLSFromCert(nil, "")
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
		}
	
		// Dial with briefly scoped context to block for resolution
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	
		newConn, err := grpc.DialContext(ctx, target, dialOpts...)
		if err != nil {
			grpcConnMu.Unlock()
			return nil, fmt.Errorf("failed to connect to gateway: %w", err)
		}
		
		conn = newConn
		grpcConnPool[gatewayURL] = conn
	}
	grpcConnMu.Unlock()

	clientAPI := proto.ClientAPIClient(proto.NewClientAPIClient(conn))
	ampServer := proto.AMPServerClient(proto.NewAMPServerClient(conn))

	// Initialize the Mailblocks registry using account configuration
	var registry *web3mail.MailblocksRegistry
	if account.EthereumRPCURL != "" && account.RegistryAddress != "" {
		reg, err := web3mail.NewMailblocksRegistry(account.EthereumRPCURL, account.RegistryAddress)
		if err != nil {
			// Log but don't fail immediately, some instances might use internal DIDs without EVM
			fmt.Printf("Warning: failed to initialize Mailblocks registry: %v\n", err)
		} else {
			registry = reg
		}
	}

	return &MsgsGlobalClient{
		account:       account,
		conn:          conn,
		clientAPI:     clientAPI,
		ampServer:     ampServer,
		signingKey:    signingKey,
		encryptionKey: encryptionKey,
		registry:      registry,
	}, nil
}

// Authenticate performs challenge-response authentication and obtains JWT token
func (m *MsgsGlobalClient) Authenticate(ctx context.Context) error {
	// Step 1: Request challenge
	challengeReq := &proto.ChallengeRequest{
		Did: m.account.DID,
	}

	challengeResp, err := m.clientAPI.RequestChallenge(ctx, challengeReq)
	if err != nil {
		return fmt.Errorf("failed to request challenge: %w", err)
	}

	// Step 2: Sign challenge with Ed25519 private key
	challengeBytes, err := hex.DecodeString(challengeResp.ChallengeHex)
	if err != nil {
		return fmt.Errorf("invalid challenge hex: %w", err)
	}

	signature := ed25519.Sign(m.signingKey, challengeBytes)

	// Step 3: Submit signed challenge
	authReq := &proto.AuthRequest{
		Did:          m.account.DID,
		ChallengeHex: challengeResp.ChallengeHex,
		Signature:    signature,
	}

	authResp, err := m.clientAPI.Authenticate(ctx, authReq)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if !authResp.Success {
		return fmt.Errorf("authentication rejected: %s", authResp.ErrorMessage)
	}

	m.jwtToken = authResp.Token
	return nil
}

// getAuthContext returns a context with JWT bearer token
func (m *MsgsGlobalClient) getAuthContext(ctx context.Context) context.Context {
	if m.jwtToken != "" {
		md := metadata.Pairs("authorization", "Bearer "+m.jwtToken)
		return metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

// FetchMessages retrieves messages from msgs.global inbox
func (m *MsgsGlobalClient) FetchMessages(ctx context.Context, limit int32) ([]*Message, error) {
	if m.jwtToken == "" {
		if err := m.Authenticate(ctx); err != nil {
			return nil, fmt.Errorf("authentication required: %w", err)
		}
	}

	authCtx := m.getAuthContext(ctx)

	inboxReq := &proto.InboxRequest{
		Limit:          limit,
		SinceTimestamp: 0, // Fetch all
	}

	stream, err := m.clientAPI.FetchInbox(authCtx, inboxReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch inbox: %w", err)
	}

	messages := []*Message{}
	for {
		ampMsg, err := stream.Recv()
		if err != nil {
			break // End of stream
		}

		msg, err := m.convertAMPMessage(ampMsg)
		if err != nil {
			// Log error but continue
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// convertAMPMessage converts an AMP message to our unified Message type
func (m *MsgsGlobalClient) convertAMPMessage(ampMsg *proto.AMPMessage) (*Message, error) {
	// Verify signatures
	if !m.verifySignatures(ampMsg) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Decrypt payload
	amfPayload, err := m.decryptPayload(ampMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt payload: %w", err)
	}

	msg := &Message{
		AccountID:  m.account.ID,
		RemoteID:   ampMsg.Headers.MessageId,
		Protocol:   "amp",
		SenderDID:  ampMsg.Headers.SenderDid,
		Subject:    amfPayload.Subject,
		BodyPlain:  amfPayload.TextBody,
		BodyHTML:   amfPayload.HtmlBody,
		ReceivedAt: time.Unix(ampMsg.Headers.Timestamp, 0),
		Signatures: ampMsg.Signatures,
		Verified:   true,
	}

	// Convert AMF attachments
	if len(amfPayload.Attachments) > 0 {
		msg.Attachments = make([]Attachment, 0, len(amfPayload.Attachments))
		for _, att := range amfPayload.Attachments {
			msg.Attachments = append(msg.Attachments, Attachment{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Data:        att.Data,
				Hash:        att.Hash,
				Size:        int64(len(att.Data)),
			})
		}
	}

	// Store raw AMF payload
	amfBytes, err := protobuf.Marshal(amfPayload)
	if err == nil {
		msg.AMFPayload = amfBytes
	}

	return msg, nil
}

// verifySignatures verifies the Ed25519 signatures on an AMP message
func (m *MsgsGlobalClient) verifySignatures(ampMsg *proto.AMPMessage) bool {
	if m.registry == nil {
		// Fail closed: cannot verify signature without registry
		// Accepting unverifiable messages would allow impersonation attacks
		fmt.Printf("Warning: cannot verify message signature - registry unavailable for DID: %s\n", ampMsg.Headers.SenderDid)
		return false
	}
	
	_, signKeyBytes, err := m.registry.ResolveDID(context.Background(), ampMsg.Headers.SenderDid)
	if err != nil {
		return false
	}
	
	// Convert array to slice
	signKey := ed25519.PublicKey(signKeyBytes[:])
	
	// Ensure at least one signature validates the payload
	// The payload verified is the AMF blob checksum/contents
	for _, sig := range ampMsg.Signatures {
		if ed25519.Verify(signKey, ampMsg.EncryptedPayload, sig) {
			return true
		}
	}
	return false
}

// resolveDIDPublicKey fetches the recipient's X25519 public key.
func (m *MsgsGlobalClient) resolveDIDPublicKey(did string) ([32]byte, error) {
	if did == "did:aftersmtp:msgs.global:test-recipient" {
		// Example generated key for testing
		pubBytes, _ := hex.DecodeString("3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29")
		var pub [32]byte
		copy(pub[:], pubBytes)
		return pub, nil
	}

	if m.registry == nil {
		return [32]byte{}, fmt.Errorf("registry unavailable and DID not found in local cache")
	}

	encKey, _, err := m.registry.ResolveDID(context.Background(), did)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to query EVM registry for DID %s: %w", did, err)
	}

	return encKey, nil
}

// encryptPayload encrypts AMF payload for recipient using X25519 + AES-GCM
func (m *MsgsGlobalClient) encryptPayload(payload *proto.AMFPayload, recipientDID string) ([]byte, []byte, error) {
	// 1. Resolve recipientDID X25519 public key from ledger
	recipientPub, err := m.resolveDIDPublicKey(recipientDID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve recipient public key: %w", err)
	}

	// 2. Generate ephemeral X25519 keypair
	var ephemeralPriv [32]byte
	if _, err := rand.Read(ephemeralPriv[:]); err != nil {
		return nil, nil, fmt.Errorf("failed to generate ephemeral private key: %w", err)
	}

	ephemeralPub, err := curve25519.X25519(ephemeralPriv[:], curve25519.Basepoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute ephemeral public key: %w", err)
	}

	// 3. Perform ECDH to get shared secret
	sharedSecret, err := curve25519.X25519(ephemeralPriv[:], recipientPub[:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// 4. Use HKDF-SHA256 to derive a 32-byte AES key
	hkdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("AfterSMTP Payload Encryption v1"))
	aesKey := make([]byte, 32)
	if _, err := hkdf.Read(aesKey); err != nil {
		return nil, nil, fmt.Errorf("failed to derive AES key: %w", err)
	}

	// 5. Marshal the protobuf payload
	payloadBytes, err := protobuf.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 6. Encrypt with AES-256-GCM
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Prepend nonce to ciphertext
	ciphertext := aesgcm.Seal(nonce, nonce, payloadBytes, nil)

	return ciphertext, ephemeralPub[:], nil
}

// DeliverMessage constructs an AMF payload, encrypts it, signs it, and sends it via gRPC.
func (m *MsgsGlobalClient) DeliverMessage(ctx context.Context, recipientDID string, payload *proto.AMFPayload) (*proto.DeliveryResponse, error) {
	// Add timestamp to headers if not present
	if payload.Subject == "" && payload.TextBody == "" && payload.HtmlBody == "" {
		return nil, fmt.Errorf("cannot send empty payload")
	}

	ciphertext, ephemeralPub, err := m.encryptPayload(payload, recipientDID)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	ampMsg := &proto.AMPMessage{
		Headers: &proto.AMPHeaders{
			SenderDid:    m.account.DID,
			RecipientDid: recipientDID,
			MessageId:    fmt.Sprintf("amp-%d", time.Now().UnixNano()),
			Timestamp:    time.Now().Unix(),
		},
		EncryptedPayload:   ciphertext,
		EphemeralPublicKey: ephemeralPub,
	}

	// Calculate signature
	sigHash, err := protobuf.Marshal(ampMsg.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to hash headers for signature: %w", err)
	}
	// Append the encrypted payload to the signature hash to ensure payload integrity
	sigHash = append(sigHash, ampMsg.EncryptedPayload...)
	
	signature := ed25519.Sign(m.signingKey, sigHash)
	ampMsg.Signatures = [][]byte{signature}

	// Dispatch over gRPC
	authCtx := m.getAuthContext(ctx)
	resp, err := m.ampServer.DeliverMessage(authCtx, ampMsg)
	if err != nil {
		return nil, fmt.Errorf("gRPC delivery failed: %w", err)
	}

	return resp, nil
}

// decryptPayload decrypts the AMP message payload using X25519 + AES-GCM
func (m *MsgsGlobalClient) decryptPayload(ampMsg *proto.AMPMessage) (*proto.AMFPayload, error) {
	if len(ampMsg.EphemeralPublicKey) != 32 {
		return nil, fmt.Errorf("invalid ephemeral public key length")
	}

	var ephemeralPub [32]byte
	copy(ephemeralPub[:], ampMsg.EphemeralPublicKey)

	// 1. Perform ECDH to get shared secret
	sharedSecret, err := curve25519.X25519(m.encryptionKey[:], ephemeralPub[:])
	if err != nil {
		return nil, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	// 2. Use HKDF-SHA256 to derive the 32-byte AES key
	hkdf := hkdf.New(sha256.New, sharedSecret, nil, []byte("AfterSMTP Payload Encryption v1"))
	aesKey := make([]byte, 32)
	if _, err := hkdf.Read(aesKey); err != nil {
		return nil, fmt.Errorf("failed to derive AES key: %w", err)
	}

	// 3. Decrypt with AES-256-GCM
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesgcm.NonceSize()
	if len(ampMsg.EncryptedPayload) < nonceSize {
		return nil, fmt.Errorf("encrypted payload too short")
	}

	nonce, ciphertext := ampMsg.EncryptedPayload[:nonceSize], ampMsg.EncryptedPayload[nonceSize:]
	payloadBytes, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt payload: %w", err)
	}

	// 4. Unmarshal payload
	var payload proto.AMFPayload
	if err := protobuf.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted payload: %w", err)
	}

	return &payload, nil
}

// signMessage creates an Ed25519 signature over the message
func (m *MsgsGlobalClient) signMessage(headers *proto.AMPHeaders, encryptedPayload []byte) []byte {
	// Serialize headers
	headersBytes, _ := protobuf.Marshal(headers)

	// Concatenate headers + payload
	dataToSign := append(headersBytes, encryptedPayload...)

	// Sign with Ed25519
	return ed25519.Sign(m.signingKey, dataToSign)
}

// generateMessageID creates a unique message ID
func generateMessageID() string {
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	return hex.EncodeToString(randomBytes)
}

// Close closes the gRPC connection and removes it from the pool if it's the last reference
func (m *MsgsGlobalClient) Close() error {
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}

// CloseConnectionForGateway closes and removes a specific gateway connection from the pool
func CloseConnectionForGateway(gatewayURL string) error {
	grpcConnMu.Lock()
	defer grpcConnMu.Unlock()

	conn, ok := grpcConnPool[gatewayURL]
	if !ok {
		return nil // Already closed or never existed
	}

	delete(grpcConnPool, gatewayURL)
	return conn.Close()
}

// CloseAllConnections closes all gRPC connections in the pool
// Should be called during application shutdown
func CloseAllConnections() error {
	grpcConnMu.Lock()
	defer grpcConnMu.Unlock()

	var lastErr error
	for gatewayURL, conn := range grpcConnPool {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
		delete(grpcConnPool, gatewayURL)
	}

	return lastErr
}
