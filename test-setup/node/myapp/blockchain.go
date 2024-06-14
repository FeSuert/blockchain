package main

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cbergoon/merkletree"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/libp2p/go-libp2p/core/host"
)

type Content struct {
	x string
}

var blockchainMutex sync.Mutex

func (t Content) Equals(other merkletree.Content) (bool, error) {
	return t.x == other.(Content).x, nil
}

func (t Content) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(t.x)); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func MerkleRootHash(data []string) ([]byte, error) {
	var list []merkletree.Content
	for _, d := range data {
		list = append(list, Content{x: d})
	}

	tree, err := merkletree.NewTree(list)
	if err != nil {
		return nil, err
	}

	return tree.MerkleRoot(), nil
}

type Block struct {
	ID           int
	PrevID       int
	LeaderValue  int
	Transactions []string
}

func SaveBlock(block Block, state *ConsensusState, config Config, node host.Host) (int, error) {
	blockchainMutex.Lock()
	defer blockchainMutex.Unlock()

	filePath := "./data/blockchain.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return state.CurrentBlockID, err
	}

	newLine := fmt.Sprintf("%d/%d/%d/%s", block.ID, block.PrevID, block.LeaderValue, strings.Join(block.Transactions, ", "))

	var newLines []string
	for _, line := range lines {
		parts := strings.SplitN(line, "/", 4)
		if len(parts) < 4 {
			continue
		}

		existingID, _ := strconv.Atoi(parts[0])
		existingLeaderValue, _ := strconv.Atoi(parts[2])

		if block.ID == existingID {
			if block.LeaderValue > existingLeaderValue || block.LeaderValue == existingLeaderValue {
				return state.CurrentBlockID, nil
			} else {
				break
			}
		}
		newLines = append(newLines, line)
	}

	newLines = append(newLines, newLine)

	for _, peer := range config.Peers {
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		connectToPeer(node, peerID, address)
		SendMessage(node, address, newLine+"\n", "/blockchain")
	}

	sort.SliceStable(newLines, func(i, j int) bool {
		id1, _ := strconv.Atoi(strings.SplitN(newLines[i], "/", 4)[0])
		id2, _ := strconv.Atoi(strings.SplitN(newLines[j], "/", 4)[0])
		return id1 < id2
	})

	if err = writeAllLines(filePath, newLines); err != nil {
		return state.CurrentBlockID, err
	}
	for _, msg := range block.Transactions {
		if err = removeTransaction(msg); err != nil {
			fmt.Println("Error removing transaction:", err)
		}
		parts := strings.SplitN(msg, "|", 2)
		tx, err := parseTransactionFromLine(parts[1])
		if err != nil {
			fmt.Println("Error parsing transaction BLOCKCHAIN:", err)
		}
		correct, err := validateTransaction(tx)
		if err != nil {
			fmt.Println("Error checking transaction:", err)
		}
		if correct {
			// Execute the transaction in the EVM
			blockCtx := vm.BlockContext{}
			txCtx := vm.TxContext{}
			evm := initializeEVM(blockCtx, txCtx, stateDB)
			_, err := executeTransaction(evm, tx)
			if err != nil {
				fmt.Println("Error executing transaction:", err)
			}
		}

	}

	if len(newLines) == 0 {
		return state.CurrentBlockID, fmt.Errorf("no new lines in blockchain")
	}

	lastID, _ := strconv.Atoi(strings.SplitN(newLines[len(newLines)-1], "/", 4)[0])
	return lastID + 1, nil
}

func CalculateLeaderValue(root []byte, config Config, nodeName string) int {
	h := sha256.New()
	h.Write(root)
	h.Write([]byte(nodeName))
	hashedRoot := h.Sum(nil)

	var sum int
	for _, b := range hashedRoot {
		sum += int(b)
	}

	leaderValue := int(float64(sum) * (1 - config.LeaderProbability))
	return leaderValue
}

func GetCurrentBlockID() (int, error) {
	filePath := "./data/blockchain.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return 0, err
	}

	if len(lines) == 0 {
		return 0, nil
	}

	lastLine := lines[len(lines)-1]
	parts := strings.SplitN(lastLine, "/", 4)
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid block format")
	}

	return strconv.Atoi(parts[0])
}
