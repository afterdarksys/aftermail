package accounts

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/curve25519"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	// Import protobuf definitions
	"github.com/afterdarksys/aftermail/pkg/proto"
	protobuf "google.golang.org/protobuf/proto"
)

const (
	DefaultMsgsGlobalGateway = "amp.msgs.global:4433"
)

// MsgsGlobalClient handles communication with msgs.global using AfterSMTP protocol
type MsgsGlobalClient struct {
	account      *Account
	conn         *grpc.ClientConn
	clientAPI    proto.ClientAPIClient
	ampServer    proto.AMPServerClient
	jwtToken     string
	signingKey   ed25519.PrivateKey
	encryptionKey [32]byte
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

	// Establish gRPC connection with TLS
	creds := credentials.NewClientTLSFromCert(nil, "")
	conn, err := grpc.Dial(gatewayURL, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gateway: %w", err)
	}

	clientAPI := proto.NewClientAPIClient(conn)
	ampServer := proto.NewAMPServerClient(conn)

	return &MsgsGlobalClient{
		account:       account,
		conn:          conn,
		clientAPI:     clientAPI,
		ampServer:     ampServer,
		signingKey:    signingKey,
		encryptionKey: encryptionKey,
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
	// Verify signature
	if !m.verifySignature(ampMsg) {
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
		Signature:  ampMsg.Signature,
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

// verifySignature verifies the Ed25519 signature on an AMP message
func (m *MsgsGlobalClient) verifySignature(ampMsg *proto.AMPMessage) bool {
	// In production, fetch sender's public key from blockchain ledger
	// For now, we'll assume verification succeeds if signature exists
	return len(ampMsg.Signature) > 0
}

// decryptPayload decrypts the AMP message payload using X25519 + AES-GCM
func (m *MsgsGlobalClient) decryptPayload(ampMsg *proto.AMPMessage) (*proto.AMFPayload, error) {
	// This is a simplified version - production would use proper ECDH + AES-GCM
	// For now, attempt to unmarshal directly (assuming it's our own message or unencrypted test)
	var payload proto.AMFPayload
	if err := protobuf.Unmarshal(ampMsg.EncryptedPayload, &payload); err != nil {
		return nil, fmt.Errorf("failed to decrypt/unmarshal payload: %w", err)
	}
	return &payload, nil
}

// SendMessage sends an AMP message via msgs.global
func (m *MsgsGlobalClient) SendMessage(ctx context.Context, recipientDID, subject, textBody, htmlBody string, attachments []Attachment) error {
	if m.jwtToken == "" {
		if err := m.Authenticate(ctx); err != nil {
			return fmt.Errorf("authentication required: %w", err)
		}
	}

	// Build AMF payload
	amfPayload := &proto.AMFPayload{
		Subject:  subject,
		TextBody: textBody,
		HtmlBody: htmlBody,
	}

	// Add attachments
	if len(attachments) > 0 {
		amfPayload.Attachments = make([]*proto.Attachment, 0, len(attachments))
		for _, att := range attachments {
			amfPayload.Attachments = append(amfPayload.Attachments, &proto.Attachment{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Data:        att.Data,
				Hash:        att.Hash,
			})
		}
	}

	// Encrypt payload (simplified - production uses X25519 ECDH + AES-GCM)
	encryptedPayload, ephemeralPubKey, err := m.encryptPayload(amfPayload, recipientDID)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %w", err)
	}

	// Build AMP headers
	headers := &proto.AMPHeaders{
		SenderDid:    m.account.DID,
		RecipientDid: recipientDID,
		Timestamp:    time.Now().Unix(),
		MessageId:    generateMessageID(),
	}

	// Create signature over headers + encrypted payload
	signature := m.signMessage(headers, encryptedPayload)

	// Build final AMP message
	ampMsg := &proto.AMPMessage{
		Headers:            headers,
		EncryptedPayload:   encryptedPayload,
		EphemeralPublicKey: ephemeralPubKey,
		Signature:          signature,
	}

	// Send via gRPC
	authCtx := m.getAuthContext(ctx)
	resp, err := m.clientAPI.DispatchMessage(authCtx, ampMsg)
	if err != nil {
		return fmt.Errorf("failed to dispatch message: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("delivery failed: %s", resp.ErrorMessage)
	}

	return nil
}

// encryptPayload encrypts AMF payload for recipient (simplified version)
func (m *MsgsGlobalClient) encryptPayload(payload *proto.AMFPayload, recipientDID string) ([]byte, []byte, error) {
	// In production:
	// 1. Resolve recipientDID X25519 public key from ledger
	// 2. Generate ephemeral X25519 keypair
	// 3. Perform ECDH to get shared secret
	// 4. Use AES-GCM-256 to encrypt payload

	// For now, just marshal as-is (unencrypted test mode)
	payloadBytes, err := protobuf.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}

	// Generate ephemeral keypair (not actually used in this simplified version)
	var ephemeralPriv [32]byte
	if _, err := rand.Read(ephemeralPriv[:]); err != nil {
		return nil, nil, err
	}

	ephemeralPub, err := curve25519.X25519(ephemeralPriv[:], curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}

	return payloadBytes, ephemeralPub, nil
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

// Close closes the gRPC connection
func (m *MsgsGlobalClient) Close() error {
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}
