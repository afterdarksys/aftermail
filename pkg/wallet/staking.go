package wallet

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// StakingProtocol represents different staking protocols
type StakingProtocol string

const (
	// Ethereum 2.0 staking
	ETH2Staking StakingProtocol = "eth2"

	// Liquid staking protocols
	LidoStaking    StakingProtocol = "lido"
	RocketPool     StakingProtocol = "rocketpool"
	StakeWiseStaking StakingProtocol = "stakewise"

	// Layer 2 staking
	PolygonStaking StakingProtocol = "polygon"
)

// StakingManager handles staking operations
type StakingManager struct {
	Wallet   *EthereumWallet
	Protocol StakingProtocol
	Contract *SmartContract
}

// StakingInfo represents staking status
type StakingInfo struct {
	TotalStaked    *big.Int
	Rewards        *big.Int
	IsStaking      bool
	ValidatorCount int
	APR            float64
}

// NewStakingManager creates a new staking manager
func NewStakingManager(wallet *EthereumWallet, protocol StakingProtocol, rpcURL string) (*StakingManager, error) {
	var contractAddress string
	var abiJSON string

	switch protocol {
	case LidoStaking:
		// Lido stETH contract on Ethereum mainnet
		contractAddress = "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"
		abiJSON = LidoStakingABI
	case RocketPool:
		// Rocket Pool rETH contract
		contractAddress = "0xae78736Cd615f374D3085123A210448E74Fc6393"
		abiJSON = RocketPoolABI
	default:
		return nil, fmt.Errorf("unsupported staking protocol: %s", protocol)
	}

	contract, err := NewSmartContract(contractAddress, abiJSON, rpcURL, wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize staking contract: %w", err)
	}

	return &StakingManager{
		Wallet:   wallet,
		Protocol: protocol,
		Contract: contract,
	}, nil
}

// Stake stakes ETH in the protocol
func (sm *StakingManager) Stake(ctx context.Context, amount *big.Int) (*types.Transaction, error) {
	switch sm.Protocol {
	case LidoStaking:
		return sm.stakeLido(ctx, amount)
	case RocketPool:
		return sm.stakeRocketPool(ctx, amount)
	default:
		return nil, fmt.Errorf("staking not implemented for protocol: %s", sm.Protocol)
	}
}

// stakeLido stakes ETH with Lido
func (sm *StakingManager) stakeLido(ctx context.Context, amount *big.Int) (*types.Transaction, error) {
	// Lido staking is done by calling submit() with ETH value
	gasLimit, err := sm.Contract.EstimateGas(ctx, "submit", amount, common.Address{})
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}

	tx, err := sm.Contract.SendTransaction(ctx, "submit", amount, gasLimit, common.Address{})
	if err != nil {
		return nil, fmt.Errorf("failed to stake: %w", err)
	}

	return tx, nil
}

// stakeRocketPool stakes ETH with Rocket Pool
func (sm *StakingManager) stakeRocketPool(ctx context.Context, amount *big.Int) (*types.Transaction, error) {
	// Rocket Pool staking via deposit
	gasLimit, err := sm.Contract.EstimateGas(ctx, "deposit", amount)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}

	tx, err := sm.Contract.SendTransaction(ctx, "deposit", amount, gasLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to stake: %w", err)
	}

	return tx, nil
}

// GetStakingInfo retrieves current staking information
func (sm *StakingManager) GetStakingInfo(ctx context.Context) (*StakingInfo, error) {
	var balance *big.Int

	switch sm.Protocol {
	case LidoStaking:
		// Get stETH balance
		if err := sm.Contract.CallMethod(ctx, &balance, "balanceOf", sm.Wallet.Address); err != nil {
			return nil, fmt.Errorf("failed to get balance: %w", err)
		}
	case RocketPool:
		// Get rETH balance
		if err := sm.Contract.CallMethod(ctx, &balance, "balanceOf", sm.Wallet.Address); err != nil {
			return nil, fmt.Errorf("failed to get balance: %w", err)
		}
	default:
		return nil, fmt.Errorf("staking info not implemented for protocol: %s", sm.Protocol)
	}

	return &StakingInfo{
		TotalStaked: balance,
		IsStaking:   balance.Cmp(big.NewInt(0)) > 0,
	}, nil
}

// GetRewards calculates accumulated staking rewards
func (sm *StakingManager) GetRewards(ctx context.Context) (*big.Int, error) {
	// This would require tracking original stake amount vs current balance
	// For liquid staking tokens, the token appreciates in value
	info, err := sm.GetStakingInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Simplified: actual implementation would track deposit amounts
	return info.TotalStaked, nil
}

// Unstake withdraws staked ETH (if supported by protocol)
func (sm *StakingManager) Unstake(ctx context.Context, amount *big.Int) (*types.Transaction, error) {
	switch sm.Protocol {
	case LidoStaking:
		// Lido requires withdrawal request via withdrawal queue
		return nil, fmt.Errorf("lido unstaking requires withdrawal queue - use RequestWithdrawal")
	case RocketPool:
		// Rocket Pool allows burning rETH for ETH
		gasLimit, err := sm.Contract.EstimateGas(ctx, "burn", nil, amount)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas: %w", err)
		}
		return sm.Contract.SendTransaction(ctx, "burn", nil, gasLimit, amount)
	default:
		return nil, fmt.Errorf("unstaking not implemented for protocol: %s", sm.Protocol)
	}
}

// Simplified ABIs for staking contracts
const LidoStakingABI = `[
	{
		"name": "submit",
		"type": "function",
		"inputs": [{"name": "_referral", "type": "address"}],
		"outputs": [{"name": "", "type": "uint256"}],
		"stateMutability": "payable"
	},
	{
		"name": "balanceOf",
		"type": "function",
		"inputs": [{"name": "_account", "type": "address"}],
		"outputs": [{"name": "", "type": "uint256"}],
		"stateMutability": "view"
	}
]`

const RocketPoolABI = `[
	{
		"name": "deposit",
		"type": "function",
		"inputs": [],
		"outputs": [],
		"stateMutability": "payable"
	},
	{
		"name": "burn",
		"type": "function",
		"inputs": [{"name": "_rethAmount", "type": "uint256"}],
		"outputs": [],
		"stateMutability": "nonpayable"
	},
	{
		"name": "balanceOf",
		"type": "function",
		"inputs": [{"name": "_account", "type": "address"}],
		"outputs": [{"name": "", "type": "uint256"}],
		"stateMutability": "view"
	}
]`
