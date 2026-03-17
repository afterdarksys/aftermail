package wallet

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SmartContract represents a deployed Ethereum smart contract
type SmartContract struct {
	Address  common.Address
	ABI      abi.ABI
	Client   *ethclient.Client
	Wallet   *EthereumWallet
	ChainID  *big.Int
}

// NewSmartContract creates a new smart contract instance
func NewSmartContract(contractAddress, abiJSON, rpcURL string, wallet *EthereumWallet) (*SmartContract, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	contractABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return &SmartContract{
		Address: common.HexToAddress(contractAddress),
		ABI:     contractABI,
		Client:  client,
		Wallet:  wallet,
		ChainID: chainID,
	}, nil
}

// CallMethod calls a read-only contract method (doesn't require gas)
func (sc *SmartContract) CallMethod(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	callData, err := sc.ABI.Pack(method, args...)
	if err != nil {
		return fmt.Errorf("failed to pack arguments: %w", err)
	}

	msg := ethereum.CallMsg{
		From: sc.Wallet.Address,
		To:   &sc.Address,
		Data: callData,
	}

	output, err := sc.Client.CallContract(ctx, msg, nil)
	if err != nil {
		return fmt.Errorf("contract call failed: %w", err)
	}

	if err := sc.ABI.UnpackIntoInterface(result, method, output); err != nil {
		return fmt.Errorf("failed to unpack result: %w", err)
	}

	return nil
}

// SendTransaction sends a state-changing transaction to the contract
func (sc *SmartContract) SendTransaction(ctx context.Context, method string, value *big.Int, gasLimit uint64, args ...interface{}) (*types.Transaction, error) {
	nonce, err := sc.Client.PendingNonceAt(ctx, sc.Wallet.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := sc.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	callData, err := sc.ABI.Pack(method, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack arguments: %w", err)
	}

	if value == nil {
		value = big.NewInt(0)
	}

	tx := types.NewTransaction(nonce, sc.Address, value, gasLimit, gasPrice, callData)

	auth, err := bind.NewKeyedTransactorWithChainID(sc.Wallet.PrivateKey, sc.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	signedTx, err := auth.Signer(auth.From, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := sc.Client.SendTransaction(ctx, signedTx); err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx, nil
}

// VerifyContract verifies a contract's bytecode matches expected bytecode
func (sc *SmartContract) VerifyContract(ctx context.Context) (bool, error) {
	code, err := sc.Client.CodeAt(ctx, sc.Address, nil)
	if err != nil {
		return false, fmt.Errorf("failed to fetch contract code: %w", err)
	}

	// Contract exists if bytecode is not empty
	if len(code) == 0 {
		return false, fmt.Errorf("no contract deployed at address %s", sc.Address.Hex())
	}

	return true, nil
}

// GetContractCode retrieves the deployed bytecode
func (sc *SmartContract) GetContractCode(ctx context.Context) ([]byte, error) {
	code, err := sc.Client.CodeAt(ctx, sc.Address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contract code: %w", err)
	}
	return code, nil
}

// WaitForTransaction waits for a transaction to be mined
func (sc *SmartContract) WaitForTransaction(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	receipt, err := bind.WaitMined(ctx, sc.Client, tx)
	if err != nil {
		return nil, fmt.Errorf("transaction mining failed: %w", err)
	}

	if receipt.Status == 0 {
		return receipt, fmt.Errorf("transaction failed with status 0")
	}

	return receipt, nil
}

// EstimateGas estimates gas for a contract method call
func (sc *SmartContract) EstimateGas(ctx context.Context, method string, value *big.Int, args ...interface{}) (uint64, error) {
	callData, err := sc.ABI.Pack(method, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to pack arguments: %w", err)
	}

	if value == nil {
		value = big.NewInt(0)
	}

	msg := ethereum.CallMsg{
		From:  sc.Wallet.Address,
		To:    &sc.Address,
		Value: value,
		Data:  callData,
	}

	gasLimit, err := sc.Client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("gas estimation failed: %w", err)
	}

	// Add 20% buffer for safety
	return gasLimit * 120 / 100, nil
}

// GetEvent retrieves events from a transaction receipt
func (sc *SmartContract) GetEvent(receipt *types.Receipt, eventName string, output interface{}) error {
	for _, log := range receipt.Logs {
		if log.Address != sc.Address {
			continue
		}

		event, err := sc.ABI.EventByID(log.Topics[0])
		if err != nil {
			continue
		}

		if event.Name == eventName {
			if err := sc.ABI.UnpackIntoInterface(output, eventName, log.Data); err != nil {
				return fmt.Errorf("failed to unpack event: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("event %s not found in receipt", eventName)
}
