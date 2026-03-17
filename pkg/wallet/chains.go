package wallet

import (
	"fmt"
	"math/big"
)

// ChainConfig represents an Ethereum network configuration
type ChainConfig struct {
	ChainID       *big.Int
	Name          string
	RPCEndpoint   string
	ExplorerURL   string
	IsLayer2      bool
	Layer2Type    string // "optimism", "arbitrum", "polygon", "base", "zksync", etc.
	NativeCurrency string
}

// Common Ethereum networks
var (
	// Layer 1 Networks
	EthereumMainnet = &ChainConfig{
		ChainID:       big.NewInt(1),
		Name:          "Ethereum Mainnet",
		RPCEndpoint:   "https://eth.llamarpc.com",
		ExplorerURL:   "https://etherscan.io",
		IsLayer2:      false,
		NativeCurrency: "ETH",
	}

	EthereumSepolia = &ChainConfig{
		ChainID:       big.NewInt(11155111),
		Name:          "Sepolia Testnet",
		RPCEndpoint:   "https://rpc.sepolia.org",
		ExplorerURL:   "https://sepolia.etherscan.io",
		IsLayer2:      false,
		NativeCurrency: "ETH",
	}

	// Layer 2 Networks
	OptimismMainnet = &ChainConfig{
		ChainID:       big.NewInt(10),
		Name:          "Optimism",
		RPCEndpoint:   "https://mainnet.optimism.io",
		ExplorerURL:   "https://optimistic.etherscan.io",
		IsLayer2:      true,
		Layer2Type:    "optimism",
		NativeCurrency: "ETH",
	}

	ArbitrumOne = &ChainConfig{
		ChainID:       big.NewInt(42161),
		Name:          "Arbitrum One",
		RPCEndpoint:   "https://arb1.arbitrum.io/rpc",
		ExplorerURL:   "https://arbiscan.io",
		IsLayer2:      true,
		Layer2Type:    "arbitrum",
		NativeCurrency: "ETH",
	}

	PolygonMainnet = &ChainConfig{
		ChainID:       big.NewInt(137),
		Name:          "Polygon PoS",
		RPCEndpoint:   "https://polygon-rpc.com",
		ExplorerURL:   "https://polygonscan.com",
		IsLayer2:      true,
		Layer2Type:    "polygon",
		NativeCurrency: "MATIC",
	}

	BaseMainnet = &ChainConfig{
		ChainID:       big.NewInt(8453),
		Name:          "Base",
		RPCEndpoint:   "https://mainnet.base.org",
		ExplorerURL:   "https://basescan.org",
		IsLayer2:      true,
		Layer2Type:    "base",
		NativeCurrency: "ETH",
	}

	ZkSyncEra = &ChainConfig{
		ChainID:       big.NewInt(324),
		Name:          "zkSync Era",
		RPCEndpoint:   "https://mainnet.era.zksync.io",
		ExplorerURL:   "https://explorer.zksync.io",
		IsLayer2:      true,
		Layer2Type:    "zksync",
		NativeCurrency: "ETH",
	}

	// All supported chains
	SupportedChains = map[int64]*ChainConfig{
		1:        EthereumMainnet,
		11155111: EthereumSepolia,
		10:       OptimismMainnet,
		42161:    ArbitrumOne,
		137:      PolygonMainnet,
		8453:     BaseMainnet,
		324:      ZkSyncEra,
	}
)

// GetChainByID returns chain config by chain ID
func GetChainByID(chainID int64) (*ChainConfig, error) {
	chain, ok := SupportedChains[chainID]
	if !ok {
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
	return chain, nil
}

// GetChainByName returns chain config by name
func GetChainByName(name string) (*ChainConfig, error) {
	for _, chain := range SupportedChains {
		if chain.Name == name {
			return chain, nil
		}
	}
	return nil, fmt.Errorf("unsupported chain: %s", name)
}
