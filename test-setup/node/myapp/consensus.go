package main

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	ethState "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/libp2p/go-libp2p/core/host"
)

type ConsensusState struct {
	CurrentBlockID         int
	ReceivedMinLeaderValue int
	OwnLeaderValue         int
	LeaderValueBlockID     int
	LeaderValues           map[int]int // Stores leader values for each block ID
}

func startConsensus(node host.Host, config Config, state *ConsensusState) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Initialize the state database (persistent)
	db := rawdb.NewMemoryDatabase()
	rootHash := common.Hash{}
	stateDB, _ := ethState.New(rootHash, ethState.NewDatabase(db), nil)

	// Initialize genesis accounts and balances
	for addr, balance := range config.Addresses {
		slicedHex := addr[2:]
		address := common.HexToAddress(slicedHex)
		stateDB.CreateAccount(address)
		stateDB.SetBalance(address, uint256.MustFromBig(big.NewInt(int64(balance))), 0)
	}

	for range ticker.C {
		// Read the current block ID from the blockchain file
		blockchainFilePath := "./data/blockchain.txt"
		blockchainLines, err := readAllLines(blockchainFilePath)
		if err != nil {
			fmt.Println("Error reading blockchain file:", err)
			continue
		}

		if len(blockchainLines) > 0 {
			lastLine := blockchainLines[len(blockchainLines)-1]
			parts := strings.SplitN(lastLine, "/", 4)
			if len(parts) >= 1 {
				state.CurrentBlockID, err = strconv.Atoi(parts[0])
				if err != nil {
					fmt.Println("Error parsing block ID:", err)
					continue
				}
				state.CurrentBlockID++
			}
		} else {
			// If the blockchain is empty, start with block ID 1
			state.CurrentBlockID = 1
		}
		// Initialize leader values map for the current block ID
		state.LeaderValues = make(map[int]int)
		state.ReceivedMinLeaderValue = int(^uint(0) >> 1) // Max int value
		time.Sleep(10 * time.Second)
		// Read the transactions from the sorted messages file
		filePath := "./data/sorted_messages.txt"
		lines, err := readAllLines(filePath)
		if err != nil {
			fmt.Println("Error reading sorted messages file:", err)
			continue
		}

		if len(lines) < config.MinedBlockSize {
			fmt.Println("Not enough transactions to create a block.")
			continue
		}

		// Collect transactions for the new block
		blockTransactions := []string{}
		for _, line := range lines[:config.MinedBlockSize] {
			parts := strings.SplitN(line, "|", 2)
			if len(parts) != 2 {
				fmt.Println("Invalid transaction line format:", line)
				continue
			}
			transactionData := parts[1]
			var tx TX
			tx, err = parseTransactionFromLine(transactionData)
			if err != nil {
				fmt.Println("Error parsing transaction CONSENSUS:", err)
				continue
			}

			// Validate the transaction
			isValid, err := validateTransaction(tx)
			if err != nil {
				fmt.Println("Error checking transaction:", err)
				continue
			}
			if !isValid {
				fmt.Println("Invalid transaction:", tx)
				continue
			}
			// Execute the transaction in the EVM
			blockCtx := vm.BlockContext{
				Coinbase:    common.HexToAddress("0x0000000000000000000000000000000000000000"), // Default coinbase address
				GasLimit:    10000000,                                                          // Default gas limit
				BlockNumber: big.NewInt(0),
			}

			txCtx := vm.TxContext{
				Origin:   common.HexToAddress("0x0000000000000000000000000000000000000000"), // Default origin address
				GasPrice: big.NewInt(0),                                                     // Assuming no gas price for simplicity
			}
			evm := initializeEVM(blockCtx, txCtx, stateDB)
			fmt.Println("1. Here")
			_, err = executeTransaction(evm, tx)
			fmt.Println("2. Here")
			if err != nil {
				fmt.Println("Error executing transaction:", err)
				continue
			}

			blockTransactions = append(blockTransactions, line)
		}
		// Calculate the Merkle root hash for the transactions
		root, err := MerkleRootHash(blockTransactions)
		if err != nil {
			fmt.Println("Error calculating Merkle root hash:", err)
			continue
		}

		// Calculate the leader value for this node
		state.OwnLeaderValue = CalculateLeaderValue(root, config, nodeName)
		state.LeaderValueBlockID = state.CurrentBlockID

		// Send leader values to all peers
		for _, peer := range config.Peers {
			re := regexp.MustCompile(`\d+`)
			id, _ := strconv.Atoi(re.FindString(peer))
			peerID, err := getPeerIDFromPublicKey(config.Miners[id-1])
			if err != nil {
				fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
				continue
			}
			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
			connectToPeer(node, peerID, address)
			SendMessage(node, address, fmt.Sprintf("%d|%d", state.CurrentBlockID, state.OwnLeaderValue)+"\n", "/consensus")
		}

		// Wait for consensus to be reached
		time.Sleep(10 * time.Second)

		// If this node has the lowest leader value, it creates a new block
		if state.ReceivedMinLeaderValue >= state.OwnLeaderValue {
			if state.CurrentBlockID == state.LeaderValueBlockID {
				newBlock := Block{
					ID:           state.CurrentBlockID,
					PrevID:       state.CurrentBlockID - 1,
					LeaderValue:  state.OwnLeaderValue,
					Transactions: blockTransactions,
				}

				// Save the new block to the blockchain and update the state
				state.CurrentBlockID, err = SaveBlock(newBlock, state, config, node)
				if err != nil {
					fmt.Println("Error saving block:", err)
					continue
				}

				// Broadcast the new block to all peers
				blockchainLines, err = readAllLines(blockchainFilePath)
				if err != nil {
					fmt.Println("Error reading blockchain file after saving new block:", err)
					continue
				}

				for _, peer := range config.Peers {
					re := regexp.MustCompile(`\d+`)
					id, _ := strconv.Atoi(re.FindString(peer))
					peerID, err := getPeerIDFromPublicKey(config.Miners[id-1])
					if err != nil {
						fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
						continue
					}
					address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
					connectToPeer(node, peerID, address)
					SendMessage(node, address, blockchainLines[len(blockchainLines)-1], "/blockchain")
				}
			}
		}
	}
}

// Function to handle incoming leader value messages
func handleLeaderValueMessage(node host.Host, state *ConsensusState, message string) {
	parts := strings.Split(message, "|")
	if len(parts) != 2 {
		fmt.Println("Invalid leader value message:", message)
		return
	}

	receivedBlockID, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("Error parsing received block ID:", err)
		return
	}

	receivedLeaderValue, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("Error parsing received leader value:", err)
		return
	}

	// Only consider leader values for the current block ID
	if receivedBlockID == state.CurrentBlockID {
		state.LeaderValues[receivedBlockID] = receivedLeaderValue
		if receivedLeaderValue < state.ReceivedMinLeaderValue {
			state.ReceivedMinLeaderValue = receivedLeaderValue
		}
	}
}
