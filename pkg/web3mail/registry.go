package web3mail

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const registryABIJSON = `[{"constant":true,"inputs":[{"name":"did","type":"string"}],"name":"resolveIdentity","outputs":[{"name":"encryptionKey","type":"bytes32"},{"name":"signingKey","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"}]`

// MailblocksRegistry connects to the Ethereum Mailblocks namespace.
type MailblocksRegistry struct {
	client   *ethclient.Client
	contract common.Address
	parsed   abi.ABI
}

func NewMailblocksRegistry(rpcURL string, contractAddress string) (*MailblocksRegistry, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(registryABIJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse registry ABI: %w", err)
	}

	return &MailblocksRegistry{
		client:   client,
		contract: common.HexToAddress(contractAddress),
		parsed:   parsedABI,
	}, nil
}

// ResolveDID queries the smart contract for the identity keys.
func (r *MailblocksRegistry) ResolveDID(ctx context.Context, did string) (encryptionKey [32]byte, signingKey [32]byte, err error) {
	callData, err := r.parsed.Pack("resolveIdentity", did)
	if err != nil {
		return [32]byte{}, [32]byte{}, fmt.Errorf("failed to pack smart contract call: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &r.contract,
		Data: callData,
	}

	output, err := r.client.CallContract(ctx, msg, nil)
	if err != nil {
		return [32]byte{}, [32]byte{}, fmt.Errorf("failed to call smart contract: %w", err)
	}

	results, err := r.parsed.Unpack("resolveIdentity", output)
	if err != nil {
		return [32]byte{}, [32]byte{}, fmt.Errorf("failed to unpack registry output: %w", err)
	}

	if len(results) != 2 {
		return [32]byte{}, [32]byte{}, fmt.Errorf("unexpected number of return values from registry")
	}

	encKey, ok1 := results[0].([32]byte)
	signKey, ok2 := results[1].([32]byte)

	if !ok1 || !ok2 {
		return [32]byte{}, [32]byte{}, fmt.Errorf("type assertion failed for registry keys")
	}

	return encKey, signKey, nil
}
