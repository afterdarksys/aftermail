package web3mail

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/afterdarksys/aftermail/pkg/proto"
)

// SmartContractListener monitors Ethereum events and triggers emails.
type SmartContractListener struct {
	client     *ethclient.Client
	rpcURL     string
	contract   common.Address
	senderDID  string
	dispatcher func(*proto.AMPMessage) error
}

func NewSmartContractListener(rpcURL, contractHex, senderDID string, dispatcher func(*proto.AMPMessage) error) (*SmartContractListener, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum RPC: %w", err)
	}

	return &SmartContractListener{
		client:     client,
		rpcURL:     rpcURL,
		contract:   common.HexToAddress(contractHex),
		senderDID:  senderDID,
		dispatcher: dispatcher,
	}, nil
}

func (l *SmartContractListener) Start(ctx context.Context) error {
	go l.runWithReconnect(ctx)
	return nil
}

// runWithReconnect handles subscription with automatic reconnection on errors
func (l *SmartContractListener) runWithReconnect(ctx context.Context) {
	const maxRetries = 10
	const initialBackoff = 1 * time.Second
	const maxBackoff = 5 * time.Minute

	retryCount := 0

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Web3Mail] Context cancelled, stopping listener for contract %s", l.contract.Hex())
			return
		default:
		}

		// Calculate exponential backoff
		if retryCount > 0 {
			backoff := time.Duration(math.Min(
				float64(initialBackoff)*math.Pow(2, float64(retryCount-1)),
				float64(maxBackoff),
			))
			log.Printf("[Web3Mail] Reconnecting in %v (attempt %d/%d)...", backoff, retryCount, maxRetries)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
		}

		// Attempt to reconnect to Ethereum client if needed
		if l.client == nil || retryCount > 0 {
			client, err := ethclient.Dial(l.rpcURL)
			if err != nil {
				log.Printf("[Web3Mail] Failed to reconnect to Ethereum RPC: %v", err)
				retryCount++
				if retryCount >= maxRetries {
					log.Printf("[Web3Mail] Max retries reached, giving up")
					return
				}
				continue
			}
			l.client = client
		}

		// Subscribe to logs
		query := ethereum.FilterQuery{
			Addresses: []common.Address{l.contract},
		}

		logs := make(chan types.Log)
		sub, err := l.client.SubscribeFilterLogs(ctx, query, logs)
		if err != nil {
			log.Printf("[Web3Mail] Failed to subscribe to contract logs: %v", err)
			retryCount++
			if retryCount >= maxRetries {
				log.Printf("[Web3Mail] Max retries reached, giving up")
				return
			}
			continue
		}

		log.Printf("[Web3Mail] Listening for events on contract %s", l.contract.Hex())
		retryCount = 0 // Reset retry count on successful connection

		// Event loop
		disconnected := false
		for !disconnected {
			select {
			case <-ctx.Done():
				sub.Unsubscribe()
				return
			case err := <-sub.Err():
				log.Printf("[Web3Mail] Subscription error: %v, will reconnect...", err)
				sub.Unsubscribe()
				disconnected = true
				retryCount = 1
			case vLog := <-logs:
				log.Printf("[Web3Mail] Event triggered at block %d! Spooling email...", vLog.BlockNumber)
				l.triggerEmail(vLog)
			}
		}
	}
}

func (l *SmartContractListener) triggerEmail(vLog types.Log) {
	// Envelope definition
	msg := &proto.AMPMessage{
		Headers: &proto.AMPHeaders{
			SenderDid: l.senderDID,
			Timestamp: time.Now().Unix(),
		},
		BlockchainProof: vLog.TxHash.Hex(),
	}

	if l.dispatcher != nil {
		if err := l.dispatcher(msg); err != nil {
			log.Printf("[Web3Mail] Dispatcher failed: %v", err)
		} else {
			log.Printf("[Web3Mail] Smart contract email successfully dispatched for Tx: %s", vLog.TxHash.Hex())
		}
	}
}
