package wallet

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// MailblocksClient interacts with Mailblocks smart contracts
type MailblocksClient struct {
	Client          *ethclient.Client
	Wallet          *EthereumWallet
	ContractAddress common.Address
	NetworkName     string
}

// NewMailblocksClient creates a new Mailblocks client
func NewMailblocksClient(rpcURL string, contractAddr string, wallet *EthereumWallet) (*MailblocksClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	return &MailblocksClient{
		Client:          client,
		Wallet:          wallet,
		ContractAddress: common.HexToAddress(contractAddr),
		NetworkName:     "sepolia", // or mainnet
	}, nil
}

// GetBalance returns the ETH balance of the wallet
func (m *MailblocksClient) GetBalance(ctx context.Context) (*big.Int, error) {
	balance, err := m.Client.BalanceAt(ctx, m.Wallet.Address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
}

// GetStakedBalance returns the staked balance in Mailblocks contract
func (m *MailblocksClient) GetStakedBalance(ctx context.Context) (*big.Int, error) {
	// TODO: Call contract method to get staked balance
	// This would require the contract ABI and binding
	return big.NewInt(0), nil
}

// StakeForEmail stakes ETH to send an email
func (m *MailblocksClient) StakeForEmail(ctx context.Context, messageID string, recipient common.Address, amount *big.Int) (string, error) {
	// Create transaction opts
	auth, err := m.createTransactor(ctx, amount)
	if err != nil {
		return "", err
	}

	// TODO: Call contract stake method
	// This requires the contract binding
	// tx, err := contract.Stake(auth, messageID, recipient)

	// For now, return a mock transaction hash
	return "0x" + messageID[:40], nil
}

// RefundStake refunds stake for accepted email
func (m *MailblocksClient) RefundStake(ctx context.Context, messageID string) (string, error) {
	auth, err := m.createTransactor(ctx, nil)
	if err != nil {
		return "", err
	}

	// TODO: Call contract refund method
	_ = auth

	return "0x" + messageID[:40], nil
}

// SlashStake slashes stake for rejected spam
func (m *MailblocksClient) SlashStake(ctx context.Context, messageID string) (string, error) {
	auth, err := m.createTransactor(ctx, nil)
	if err != nil {
		return "", err
	}

	// TODO: Call contract slash method
	_ = auth

	return "0x" + messageID[:40], nil
}

// GetQuarantinedEmails returns emails awaiting review
func (m *MailblocksClient) GetQuarantinedEmails(ctx context.Context) ([]QuarantinedEmail, error) {
	// TODO: Query contract or backend API for quarantined emails
	return []QuarantinedEmail{}, nil
}

// QuarantinedEmail represents an email in quarantine
type QuarantinedEmail struct {
	MessageID   string
	Sender      common.Address
	Recipient   common.Address
	StakeAmount *big.Int
	IPFSCID     string
	Timestamp   int64
}

// createTransactor creates transaction options
func (m *MailblocksClient) createTransactor(ctx context.Context, value *big.Int) (*bind.TransactOpts, error) {
	chainID, err := m.Client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	nonce, err := m.Client.PendingNonceAt(ctx, m.Wallet.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := m.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(m.Wallet.PrivateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = value
	auth.GasLimit = uint64(300000)
	auth.GasPrice = gasPrice

	return auth, nil
}

// Close closes the Ethereum client connection
func (m *MailblocksClient) Close() {
	m.Client.Close()
}
