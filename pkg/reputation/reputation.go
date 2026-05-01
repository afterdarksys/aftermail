// Package reputation implements the AfterMail DID reputation system.
//
// Each sender builds a verifiable trust score from signed commitment receipts.
// When you keep a promise (the counterparty signs a "commitment-kept" receipt),
// that receipt is anchored to your DID document on itz.agency.
//
// Score components:
//   - CommitmentsKept / CommitmentsMade ratio (reliability)
//   - SignedReceipts    — Ed25519-attested by counterparties, tamper-proof
//   - StakeHistory      — consistent use of staked-attention signals skin-in-game
//   - AgeWeightedScore  — older receipts decay to prevent gaming by sudden burst
//
// The DID document is published as a TXT record on itz.agency:
//
//	_aftermail.ryan.e.tz.agency TXT "did=did:aftersmtp:msgs.global:ryan;score=0.92;receipts=47"
package reputation

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

// Receipt is a cryptographically-signed attestation that a commitment was kept.
type Receipt struct {
	// ID is a unique receipt identifier.
	ID string `json:"id"`

	// CommitmentID links back to the original commitment in the Commitment Ledger.
	CommitmentID string `json:"commitment_id"`

	// Subject is the email subject or thread reference.
	Subject string `json:"subject"`

	// Keeper is the DID of the person who kept the commitment.
	Keeper string `json:"keeper"`

	// Attester is the DID of the counterparty signing the receipt.
	Attester string `json:"attester"`

	// AttesterPublicKey is the attester's Ed25519 public key (hex).
	AttesterPublicKey string `json:"attester_public_key"`

	// IssuedAt is when the receipt was signed.
	IssuedAt time.Time `json:"issued_at"`

	// Signature is the Ed25519 signature over the canonical receipt fields.
	Signature string `json:"signature"`

	// Weight is the trust contribution of this receipt (0.0–1.0).
	// Decays over time; starts at 1.0.
	Weight float64 `json:"weight"`
}

// DIDDocument is the AfterMail reputation identity document.
type DIDDocument struct {
	// DID follows the did:aftersmtp method: did:aftersmtp:msgs.global:username
	DID string `json:"@id"`

	// Controller is the DID that controls this document.
	Controller string `json:"controller"`

	// PublicKey is the owner's Ed25519 verification key (hex).
	PublicKey string `json:"public_key"`

	// Score is the computed trust score 0.0–1.0.
	Score float64 `json:"score"`

	// ScoreBreakdown is the per-component score detail.
	ScoreBreakdown ScoreBreakdown `json:"score_breakdown"`

	// Receipts are the signed commitment-kept attestations.
	Receipts []*Receipt `json:"receipts"`

	// TotalCommitments is the total number of commitments tracked.
	TotalCommitments int `json:"total_commitments"`

	// KeptCommitments is the number that were verifiably kept.
	KeptCommitments int `json:"kept_commitments"`

	// StakeHistory is the count of staked-attention messages sent.
	StakeHistory int `json:"stake_history"`

	// UpdatedAt is the last time this document was recomputed.
	UpdatedAt time.Time `json:"updated_at"`
}

// ScoreBreakdown shows how each component contributes to the trust score.
type ScoreBreakdown struct {
	ReliabilityScore float64 `json:"reliability"` // commitments kept ratio
	ReceiptScore     float64 `json:"receipts"`    // weighted receipt density
	StakeScore       float64 `json:"stake"`       // stake usage ratio
	AgeScore         float64 `json:"age"`         // longevity bonus
}

// TrustLevel is a human-readable tier derived from Score.
type TrustLevel string

const (
	TrustUnknown   TrustLevel = "unknown"
	TrustLow       TrustLevel = "low"
	TrustEstablished TrustLevel = "established"
	TrustHigh      TrustLevel = "high"
	TrustVerified  TrustLevel = "verified"
)

// HalfLife is the time for a receipt's weight to decay to 0.5.
const HalfLife = 180 * 24 * time.Hour // 6 months

// Profile manages a local reputation profile and verifies incoming receipts.
type Profile struct {
	doc        *DIDDocument
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewProfile creates a Profile for the given DID.
func NewProfile(did string, privateKey ed25519.PrivateKey) *Profile {
	pub := privateKey.Public().(ed25519.PublicKey)
	return &Profile{
		doc: &DIDDocument{
			DID:       did,
			PublicKey: hex.EncodeToString(pub),
		},
		privateKey: privateKey,
		publicKey:  pub,
	}
}

// LoadProfile restores a Profile from a persisted DIDDocument.
func LoadProfile(doc *DIDDocument, privateKey ed25519.PrivateKey) *Profile {
	pub := privateKey.Public().(ed25519.PublicKey)
	return &Profile{doc: doc, privateKey: privateKey, publicKey: pub}
}

// IssueReceipt signs a commitment-kept receipt on behalf of this profile.
// Call this when a counterparty has demonstrably kept their commitment.
func (p *Profile) IssueReceipt(commitmentID, subject, keeperDID string) (*Receipt, error) {
	receipt := &Receipt{
		ID:                fmt.Sprintf("rcpt-%s-%d", keeperDID, time.Now().UnixNano()),
		CommitmentID:      commitmentID,
		Subject:           subject,
		Keeper:            keeperDID,
		Attester:          p.doc.DID,
		AttesterPublicKey: hex.EncodeToString(p.publicKey),
		IssuedAt:          time.Now().UTC(),
		Weight:            1.0,
	}

	payload, err := receiptPayload(receipt)
	if err != nil {
		return nil, err
	}
	receipt.Signature = hex.EncodeToString(ed25519.Sign(p.privateKey, payload))
	return receipt, nil
}

// AddReceipt verifies and adds a receipt to this profile's document.
func (p *Profile) AddReceipt(r *Receipt) error {
	if err := VerifyReceipt(r); err != nil {
		return fmt.Errorf("receipt verification failed: %w", err)
	}
	p.doc.Receipts = append(p.doc.Receipts, r)
	p.doc.KeptCommitments++
	p.Recompute()
	return nil
}

// RecordCommitment increments the total commitments counter.
func (p *Profile) RecordCommitment() {
	p.doc.TotalCommitments++
	p.Recompute()
}

// RecordStake increments the stake usage counter.
func (p *Profile) RecordStake() {
	p.doc.StakeHistory++
	p.Recompute()
}

// Recompute updates the score from current receipts and counters.
func (p *Profile) Recompute() {
	now := time.Now()
	doc := p.doc

	// 1. Reliability: kept/total ratio.
	var reliability float64
	if doc.TotalCommitments > 0 {
		reliability = float64(doc.KeptCommitments) / float64(doc.TotalCommitments)
	}

	// 2. Receipt score: sum of time-decayed weights, normalised.
	var weightedSum float64
	for _, r := range doc.Receipts {
		age := now.Sub(r.IssuedAt)
		decay := math.Pow(0.5, float64(age)/float64(HalfLife))
		weightedSum += decay
	}
	// Sigmoid-normalise: 20 receipts → score ~0.95.
	receiptScore := 1 - 1/(1+weightedSum/20)

	// 3. Stake score: sigmoid over stake count.
	stakeScore := 1 - 1/(1+float64(doc.StakeHistory)/10)

	// 4. Age score: bonus for profiles older than 6 months.
	var ageScore float64
	if !doc.UpdatedAt.IsZero() {
		age := now.Sub(doc.UpdatedAt)
		ageScore = math.Min(1.0, float64(age)/float64(HalfLife))
	}

	// Weighted combination.
	score := reliability*0.40 + receiptScore*0.35 + stakeScore*0.15 + ageScore*0.10

	doc.Score = math.Round(score*1000) / 1000
	doc.ScoreBreakdown = ScoreBreakdown{
		ReliabilityScore: math.Round(reliability*1000) / 1000,
		ReceiptScore:     math.Round(receiptScore*1000) / 1000,
		StakeScore:       math.Round(stakeScore*1000) / 1000,
		AgeScore:         math.Round(ageScore*1000) / 1000,
	}
	doc.UpdatedAt = now
}

// Document returns the current DID document.
func (p *Profile) Document() *DIDDocument { return p.doc }

// TrustLevel returns the human-readable trust tier.
func (p *Profile) TrustLevel() TrustLevel {
	return ScoreToTrustLevel(p.doc.Score)
}

// DNSTXTRecord returns the TXT record value to publish on itz.agency.
// Set this as: _aftermail.<username>.e.tz.agency TXT "<value>"
func (p *Profile) DNSTXTRecord() string {
	doc := p.doc
	return fmt.Sprintf("did=%s;score=%.3f;receipts=%d;level=%s",
		doc.DID, doc.Score, len(doc.Receipts), p.TrustLevel())
}

// VerifyReceipt checks the Ed25519 signature on a receipt.
func VerifyReceipt(r *Receipt) error {
	pubBytes, err := hex.DecodeString(r.AttesterPublicKey)
	if err != nil {
		return fmt.Errorf("invalid attester public key: %w", err)
	}
	sigBytes, err := hex.DecodeString(r.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %w", err)
	}
	payload, err := receiptPayload(r)
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), payload, sigBytes) {
		return fmt.Errorf("signature invalid")
	}
	return nil
}

// ScoreToTrustLevel converts a numeric score to a trust tier.
func ScoreToTrustLevel(score float64) TrustLevel {
	switch {
	case score >= 0.90:
		return TrustVerified
	case score >= 0.75:
		return TrustHigh
	case score >= 0.50:
		return TrustEstablished
	case score >= 0.20:
		return TrustLow
	default:
		return TrustUnknown
	}
}

// TopReceipts returns the n most recent receipts sorted by issue date.
func (doc *DIDDocument) TopReceipts(n int) []*Receipt {
	sorted := make([]*Receipt, len(doc.Receipts))
	copy(sorted, doc.Receipts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].IssuedAt.After(sorted[j].IssuedAt)
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// MarshalDocument serialises the DIDDocument to JSON.
func MarshalDocument(doc *DIDDocument) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

// UnmarshalDocument deserialises a DIDDocument from JSON.
func UnmarshalDocument(b []byte) (*DIDDocument, error) {
	var doc DIDDocument
	return &doc, json.Unmarshal(b, &doc)
}

// receiptPayload produces the canonical bytes to sign/verify.
func receiptPayload(r *Receipt) ([]byte, error) {
	type signable struct {
		ID                string `json:"id"`
		CommitmentID      string `json:"commitment_id"`
		Subject           string `json:"subject"`
		Keeper            string `json:"keeper"`
		Attester          string `json:"attester"`
		AttesterPublicKey string `json:"attester_public_key"`
		IssuedAt          string `json:"issued_at"`
	}
	s := signable{
		ID:                r.ID,
		CommitmentID:      r.CommitmentID,
		Subject:           r.Subject,
		Keeper:            r.Keeper,
		Attester:          r.Attester,
		AttesterPublicKey: r.AttesterPublicKey,
		IssuedAt:          r.IssuedAt.UTC().Format(time.RFC3339),
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	return h[:], nil
}
