package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cbergoon/merkletree"
	"github.com/libp2p/go-libp2p/core/network"
)

type Block struct {
	id           int
	prev_id      int
	leader_value int
	messages     []string
}

type Content struct {
	x string
}

func (t Content) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(t.x)); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func (t Content) Equals(other merkletree.Content) (bool, error) {
	return t.x == other.(Content).x, nil
}

func MerkleRootHash(data []string) ([]byte, error) {
	// Convert the slice of strings to a slice of Content
	var list []merkletree.Content
	for _, d := range data {
		list = append(list, Content{x: d})
	}

	// Create a new Merkle Tree from the list of Content
	tree, err := merkletree.NewTree(list)
	if err != nil {
		return nil, err
	}

	// Get the Merkle Root
	root := tree.MerkleRoot()
	return root, nil
}

func CalculateLeaderValue(root []byte) int {
	// Hash the root with the nodeName
	h := sha256.New()
	h.Write(root)
	h.Write([]byte(nodeName))
	hashedRoot := h.Sum(nil)

	// Calculate the sum of the hashed bytes
	var sum int
	for _, b := range hashedRoot {
		sum += int(b)
	}

	// Calculate the leader value
	leaderValue := int(float64(sum) * (1 - config.LeaderProbability))
	return leaderValue
}

func handleBlockchainMessage(s network.Stream) {
	reader := bufio.NewReader(s)
	receivedString, err := reader.ReadString('\n')
	receivedString = strings.TrimSpace(receivedString)
	if err != nil {
		fmt.Println("Error reading incoming string:", err)
		return
	}

	parts := strings.SplitN(receivedString, "/", 4)
	if len(parts) != 4 {
		fmt.Println("Invalid input format:", receivedString)
		return
	}
	currentID, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	prevID, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	leaderValue, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	transactions := strings.Split(strings.TrimSpace(parts[3]), ", ")

	// Generate new block
	receivedBlock := Block{
		id:           currentID,
		prev_id:      prevID,
		leader_value: leaderValue,
		messages:     transactions,
	}

	// Update file
	state.currentBlockID, err = SaveBlock(receivedBlock)
	if err != nil {
		fmt.Println("Error updating file:", err)
		return
	}
	var filePath = "./data/sorted_messages.txt"
	var blockTransactions []string
	lines, _ := readAllLines(filePath)
	for i := 0; i < config.MinedBlockSize; i++ {
		blockTransactions = append(blockTransactions, lines[i])
	}
	var root, _ = MerkleRootHash(blockTransactions)
	state.ownLeaderValue = CalculateLeaderValue(root)
}

func SaveBlock(block Block) (int, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Read the current contents of the file
	filePath := "./data/blockchain.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return state.currentBlockID, err
	}

	// Format the new message in the correct format
	newLine := fmt.Sprintf("%d/%d/%d/%s", block.id, block.prev_id, block.leader_value, strings.Join(block.messages, ", "))

	var newLines []string

	for _, line := range lines {
		parts := strings.SplitN(line, "/", 4)
		if len(parts) < 4 {
			continue
		}

		existingID, _ := strconv.Atoi(parts[0])
		existingLeaderValue, _ := strconv.Atoi(parts[2])

		if block.id == existingID {
			if block.leader_value > existingLeaderValue || block.leader_value == existingLeaderValue {
				// New block has higher leader_value
				return state.currentBlockID, nil
			} else {
				// New block has lower leader_value, remove all further blocks with higher IDs
				break
			}
		}
		newLines = append(newLines, line)
	}

	// Append the new message if it's unique or replaced an existing one
	newLines = append(newLines, newLine)

	// Send the new block to all peers
	for _, peer := range config.Peers {
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		connectToPeer(node, peerID, address)
		SendMessage(node, address, newLine+"\n", "/blockchain")
	}

	// Sort the lines based on id
	sort.SliceStable(newLines, func(i, j int) bool {
		id1, _ := strconv.Atoi(strings.SplitN(newLines[i], "/", 4)[0])
		id2, _ := strconv.Atoi(strings.SplitN(newLines[j], "/", 4)[0])
		return id1 < id2
	})

	// Write the sorted lines back to the file
	err = writeAllLines(filePath, newLines)
	if err != nil {
		fmt.Println("Writing all lines:", err)
		return state.currentBlockID, err
	}

	// Remove transactions included in the block from sorted_messages.txt
	for _, msg := range block.messages {
		err := removeTransaction(msg)
		if err != nil {
			fmt.Println("Error removing transaction:", err)
		}
	}

	// Return the last ID in the saved file
	lastID, _ := strconv.Atoi(strings.SplitN(newLines[len(newLines)-1], "/", 4)[0])
	return lastID + 1, err
}
