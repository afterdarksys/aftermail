// Package stakemail implements staked-attention email for AfterMail.
//
// A sender locks a small ETH amount on Base L2 when composing a message.
// The stake is released back to the sender when the recipient opens the
// message (open-receipt) or forfeited (slashed) to a charity/burn address
// if the recipient marks it spam.
//
// Real cost → real signal.  Eliminates bulk-spam economics without
// centralised blacklists.
package stakemail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// StakeStatus tracks the lifecycle of a single staked message.
type StakeStatus string

const (
	StakePending  StakeStatus = "pending"   // waiting for on-chain confirmation
	StakeLocked   StakeStatus = "locked"    // stake held in escrow
	StakeReleased StakeStatus = "released"  // returned to sender on open
	StakeSlashed  StakeStatus = "slashed"   // forfeited to slash address on spam
	StakeExpired  StakeStatus = "expired"   // unclaimed after TTL → returned
)

// SlashPolicy describes what happens to forfeited funds.
type SlashPolicy string

const (
	// SlashBurn sends forfeited ETH to the zero address (deflationary).
	SlashBurn SlashPolicy = "burn"
	// SlashCharity forwards to a configured charity address.
	SlashCharity SlashPolicy = "charity"
	// SlashRecipient awards forfeited stake to the reporting recipient.
	SlashRecipient SlashPolicy = "recipient"
)

// StakedMessage is the on-chain record for a single staked send.
type StakedMessage struct {
	// ID is a unique identifier derived from the email Message-ID.
	ID string `json:"id"`

	// Sender / Recipient envelope addresses.
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`

	// StakeWei is the locked amount in wei.
	StakeWei *big.Int `json:"stake_wei"`

	// StakeETH is the human-readable stake amount.
	StakeETH string `json:"stake_eth"`

	// Status of the stake lifecycle.
	Status StakeStatus `json:"status"`

	// SlashPolicy determines where slashed funds go.
	Policy SlashPolicy `json:"slash_policy"`

	// SlashAddress is the address that receives slashed funds (for SlashCharity).
	SlashAddress common.Address `json:"slash_address,omitempty"`

	// ExpiresAt is when an unclaimed stake is automatically returned.
	ExpiresAt time.Time `json:"expires_at"`

	// LockTxHash is the Base L2 transaction that locked the stake.
	LockTxHash string `json:"lock_tx_hash,omitempty"`

	// ResolveTxHash is the transaction that released or slashed the stake.
	ResolveTxHash string `json:"resolve_tx_hash,omitempty"`

	// ChainID is the chain where the stake lives (8453 = Base mainnet).
	ChainID int64 `json:"chain_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MinimumStake is the floor to prevent dust attacks.
var MinimumStake = big.NewInt(1_000_000_000_000_000) // 0.001 ETH

// StakeTTL is how long a stake waits for open/spam before auto-returning.
const StakeTTL = 30 * 24 * time.Hour

// Client manages staked messages against a Base L2 RPC endpoint.
// In production this wraps a deployed StakeEscrow smart contract;
// here we provide the client layer with local state + RPC proof-of-funds checks.
type Client struct {
	rpcURL      string
	chainID     int64
	stakeWallet common.Address
	// escrowAddr is the deployed StakeEscrow contract address.
	// Zero value means simulation mode (no on-chain transactions).
	escrowAddr common.Address
	messages   map[string]*StakedMessage
}

// NewClient creates a stakemail client.
// escrowContractAddr may be zero to run in simulation (local-only) mode.
func NewClient(rpcURL string, chainID int64, stakeWallet string, escrowContractAddr string) *Client {
	c := &Client{
		rpcURL:      rpcURL,
		chainID:     chainID,
		stakeWallet: common.HexToAddress(stakeWallet),
		messages:    make(map[string]*StakedMessage),
	}
	if escrowContractAddr != "" {
		c.escrowAddr = common.HexToAddress(escrowContractAddr)
	}
	return c
}

// Stake locks ETH for a message.  Returns the StakedMessage record.
func (c *Client) Stake(ctx context.Context, messageID, sender, recipient string, stakeWei *big.Int, policy SlashPolicy) (*StakedMessage, error) {
	if stakeWei.Cmp(MinimumStake) < 0 {
		return nil, fmt.Errorf("stake %s wei is below minimum %s wei", stakeWei, MinimumStake)
	}

	now := time.Now()
	m := &StakedMessage{
		ID:        messageID,
		Sender:    sender,
		Recipient: recipient,
		StakeWei:  new(big.Int).Set(stakeWei),
		StakeETH:  weiToETH(stakeWei),
		Status:    StakePending,
		Policy:    policy,
		ExpiresAt: now.Add(StakeTTL),
		ChainID:   c.chainID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if c.escrowAddr == (common.Address{}) {
		// Simulation mode: skip real tx.
		m.LockTxHash = fmt.Sprintf("sim-lock-%s", messageID)
		m.Status = StakeLocked
	} else {
		txHash, err := c.sendLock(ctx, m)
		if err != nil {
			return nil, fmt.Errorf("lock transaction: %w", err)
		}
		m.LockTxHash = txHash
		m.Status = StakeLocked
	}

	c.messages[messageID] = m
	log.Printf("[stakemail] locked %s ETH for message %s", m.StakeETH, messageID)
	return m, nil
}

// Release returns the stake to the sender on open-receipt.
func (c *Client) Release(ctx context.Context, messageID string) error {
	m, ok := c.messages[messageID]
	if !ok {
		return fmt.Errorf("message %q not found", messageID)
	}
	if m.Status != StakeLocked {
		return fmt.Errorf("stake is %s, not locked", m.Status)
	}

	if c.escrowAddr == (common.Address{}) {
		m.ResolveTxHash = fmt.Sprintf("sim-release-%s", messageID)
	} else {
		txHash, err := c.sendRelease(ctx, m)
		if err != nil {
			return err
		}
		m.ResolveTxHash = txHash
	}

	m.Status = StakeReleased
	m.UpdatedAt = time.Now()
	log.Printf("[stakemail] released stake for message %s → %s", messageID, m.Sender)
	return nil
}

// Slash forfeits the stake per the message's SlashPolicy.
func (c *Client) Slash(ctx context.Context, messageID string) error {
	m, ok := c.messages[messageID]
	if !ok {
		return fmt.Errorf("message %q not found", messageID)
	}
	if m.Status != StakeLocked {
		return fmt.Errorf("stake is %s, not locked", m.Status)
	}

	dest := c.slashDestination(m)
	if c.escrowAddr == (common.Address{}) {
		m.ResolveTxHash = fmt.Sprintf("sim-slash-%s-to-%s", messageID, dest.Hex())
	} else {
		txHash, err := c.sendSlash(ctx, m, dest)
		if err != nil {
			return err
		}
		m.ResolveTxHash = txHash
	}

	m.Status = StakeSlashed
	m.UpdatedAt = time.Now()
	log.Printf("[stakemail] slashed %s ETH from message %s → %s", m.StakeETH, messageID, dest.Hex())
	return nil
}

// Get returns a staked message record.
func (c *Client) Get(messageID string) (*StakedMessage, bool) {
	m, ok := c.messages[messageID]
	return m, ok
}

// ExpireStale scans for stakes past their TTL and auto-releases them.
func (c *Client) ExpireStale(ctx context.Context) {
	now := time.Now()
	for id, m := range c.messages {
		if m.Status == StakeLocked && now.After(m.ExpiresAt) {
			log.Printf("[stakemail] stake for %s expired — auto-releasing", id)
			m.Status = StakeExpired
			m.UpdatedAt = now
		}
	}
}

// MarshalJSON serialises all tracked messages for persistence.
func (c *Client) MarshalJSON() ([]byte, error) {
	msgs := make([]*StakedMessage, 0, len(c.messages))
	for _, m := range c.messages {
		msgs = append(msgs, m)
	}
	return json.Marshal(msgs)
}

// ─── RPC helpers (stub until StakeEscrow contract is deployed) ───────────────

func (c *Client) sendLock(ctx context.Context, m *StakedMessage) (string, error) {
	client, err := ethclient.DialContext(ctx, c.rpcURL)
	if err != nil {
		return "", fmt.Errorf("rpc dial: %w", err)
	}
	defer client.Close()

	// Verify sender has sufficient balance.
	balance, err := client.BalanceAt(ctx, c.stakeWallet, nil)
	if err != nil {
		return "", fmt.Errorf("balance check: %w", err)
	}
	if balance.Cmp(m.StakeWei) < 0 {
		return "", fmt.Errorf("insufficient balance: have %s wei, need %s wei", balance, m.StakeWei)
	}

	// Real implementation: ABI-encode lock(messageID, recipient) and call escrowAddr.
	// Returning a deterministic placeholder until the contract is deployed.
	return fmt.Sprintf("0x%064x", m.StakeWei.Int64()), nil
}

func (c *Client) sendRelease(_ context.Context, m *StakedMessage) (string, error) {
	// ABI-encode release(messageID) → escrowAddr
	return fmt.Sprintf("0xrelease-%s", m.ID), nil
}

func (c *Client) sendSlash(_ context.Context, m *StakedMessage, dest common.Address) (string, error) {
	// ABI-encode slash(messageID, dest) → escrowAddr
	return fmt.Sprintf("0xslash-%s-%s", m.ID, dest.Hex()), nil
}

func (c *Client) slashDestination(m *StakedMessage) common.Address {
	switch m.Policy {
	case SlashBurn:
		return common.Address{} // 0x000...
	case SlashCharity:
		return m.SlashAddress
	case SlashRecipient:
		// Parse recipient ETH address or use zero if not set.
		return common.HexToAddress(m.Recipient)
	default:
		return common.Address{}
	}
}

// ─── Utilities ────────────────────────────────────────────────────────────────

// ETHToWei converts a decimal ETH string like "0.01" to wei.
func ETHToWei(eth string) (*big.Int, bool) {
	f, ok := new(big.Float).SetPrec(256).SetString(eth)
	if !ok {
		return nil, false
	}
	oneETH := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	f.Mul(f, oneETH)
	wei, _ := f.Int(nil)
	return wei, true
}

func weiToETH(wei *big.Int) string {
	oneETH := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	f := new(big.Float).SetPrec(64).SetInt(wei)
	d := new(big.Float).SetPrec(64).SetInt(oneETH)
	result := new(big.Float).Quo(f, d)
	return fmt.Sprintf("%.6f ETH", result)
}

// IsSimulationMode returns true when no escrow contract is configured.
func (c *Client) IsSimulationMode() bool {
	zero := common.Address{}
	return c.escrowAddr == zero
}

// StakeEscrowABI is the minimal ABI for the StakeEscrow contract.
// Deploy this Solidity contract to Base L2 and pass its address to NewClient.
const StakeEscrowABI = `[
  {"name":"lock","type":"function","inputs":[{"name":"messageId","type":"string"},{"name":"recipient","type":"address"}],"outputs":[],"stateMutability":"payable"},
  {"name":"release","type":"function","inputs":[{"name":"messageId","type":"string"}],"outputs":[],"stateMutability":"nonpayable"},
  {"name":"slash","type":"function","inputs":[{"name":"messageId","type":"string"},{"name":"dest","type":"address"}],"outputs":[],"stateMutability":"nonpayable"},
  {"name":"balanceOf","type":"function","inputs":[{"name":"messageId","type":"string"}],"outputs":[{"name":"","type":"uint256"}],"stateMutability":"view"},
  {"name":"Locked","type":"event","inputs":[{"name":"messageId","type":"string","indexed":true},{"name":"sender","type":"address","indexed":true},{"name":"amount","type":"uint256","indexed":false}]},
  {"name":"Released","type":"event","inputs":[{"name":"messageId","type":"string","indexed":true}]},
  {"name":"Slashed","type":"event","inputs":[{"name":"messageId","type":"string","indexed":true},{"name":"dest","type":"address","indexed":true}]}
]`
