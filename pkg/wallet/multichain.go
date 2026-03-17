package wallet

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// MultiChainWallet manages a single wallet across multiple chains
type MultiChainWallet struct {
	Wallet  *EthereumWallet
	Clients map[int64]*ethclient.Client // chainID -> client
}

// NewMultiChainWallet creates a wallet that works across multiple chains
func NewMultiChainWallet(wallet *EthereumWallet) *MultiChainWallet {
	return &MultiChainWallet{
		Wallet:  wallet,
		Clients: make(map[int64]*ethclient.Client),
	}
}

// ConnectToChain connects to a specific chain
func (mw *MultiChainWallet) ConnectToChain(ctx context.Context, chainID int64) error {
	chain, err := GetChainByID(chainID)
	if err != nil {
		return err
	}

	client, err := ethclient.Dial(chain.RPCEndpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", chain.Name, err)
	}

	// Verify chain ID matches
	actualChainID, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify chain ID: %w", err)
	}

	if actualChainID.Int64() != chainID {
		return fmt.Errorf("chain ID mismatch: expected %d, got %d", chainID, actualChainID.Int64())
	}

	mw.Clients[chainID] = client
	return nil
}

// GetBalance returns balance on a specific chain
func (mw *MultiChainWallet) GetBalance(ctx context.Context, chainID int64) (*big.Int, error) {
	client, ok := mw.Clients[chainID]
	if !ok {
		if err := mw.ConnectToChain(ctx, chainID); err != nil {
			return nil, err
		}
		client = mw.Clients[chainID]
	}

	balance, err := client.BalanceAt(ctx, mw.Wallet.Address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

// SendETH sends native currency on a specific chain
func (mw *MultiChainWallet) SendETH(ctx context.Context, chainID int64, to common.Address, amount *big.Int) (*types.Transaction, error) {
	client, ok := mw.Clients[chainID]
	if !ok {
		if err := mw.ConnectToChain(ctx, chainID); err != nil {
			return nil, err
		}
		client = mw.Clients[chainID]
	}

	nonce, err := client.PendingNonceAt(ctx, mw.Wallet.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	gasLimit := uint64(21000) // Standard ETH transfer

	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)

	chainIDBig := big.NewInt(chainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainIDBig), mw.Wallet.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx, nil
}

// GetAllBalances returns balances across all connected chains
func (mw *MultiChainWallet) GetAllBalances(ctx context.Context) (map[string]*big.Int, error) {
	balances := make(map[string]*big.Int)

	for chainID, client := range mw.Clients {
		balance, err := client.BalanceAt(ctx, mw.Wallet.Address, nil)
		if err != nil {
			continue // Skip chains with errors
		}

		chain, err := GetChainByID(chainID)
		if err != nil {
			continue
		}

		balances[chain.Name] = balance
	}

	return balances, nil
}

// GetTransactionHistory retrieves transaction history on a chain
// Note: This is a simplified version - production would use block explorers or indexers
func (mw *MultiChainWallet) GetTransactionCount(ctx context.Context, chainID int64) (uint64, error) {
	client, ok := mw.Clients[chainID]
	if !ok {
		if err := mw.ConnectToChain(ctx, chainID); err != nil {
			return 0, err
		}
		client = mw.Clients[chainID]
	}

	count, err := client.NonceAt(ctx, mw.Wallet.Address, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get transaction count: %w", err)
	}

	return count, nil
}

// EstimateBridgeCost estimates the cost to bridge assets between chains
// This is a placeholder - actual implementation would integrate with bridge protocols
func (mw *MultiChainWallet) EstimateBridgeCost(ctx context.Context, fromChain, toChain int64, amount *big.Int) (*big.Int, error) {
	// Placeholder for bridge cost estimation
	// In production, this would integrate with bridges like:
	// - Hop Protocol
	// - Across Protocol
	// - Stargate
	// - Native bridges (Optimism/Arbitrum)

	fromChainConfig, err := GetChainByID(fromChain)
	if err != nil {
		return nil, err
	}

	toChainConfig, err := GetChainByID(toChain)
	if err != nil {
		return nil, err
	}

	// Different bridge costs for different chain combinations
	if fromChainConfig.IsLayer2 && toChainConfig.IsLayer2 {
		// L2 to L2 bridging (usually cheaper via Hop/Across)
		return big.NewInt(5000000000000000), nil // ~0.005 ETH
	}

	if !fromChainConfig.IsLayer2 && toChainConfig.IsLayer2 {
		// L1 to L2 (deposit, relatively cheap)
		return big.NewInt(10000000000000000), nil // ~0.01 ETH
	}

	if fromChainConfig.IsLayer2 && !toChainConfig.IsLayer2 {
		// L2 to L1 (withdrawal, expensive due to fraud proofs)
		return big.NewInt(50000000000000000), nil // ~0.05 ETH
	}

	return big.NewInt(20000000000000000), nil // ~0.02 ETH default
}

// Close closes all chain connections
func (mw *MultiChainWallet) Close() {
	for _, client := range mw.Clients {
		client.Close()
	}
}
