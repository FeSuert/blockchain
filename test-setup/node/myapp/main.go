package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cbergoon/merkletree"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	prt "github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/ybbus/jsonrpc"
)

var (
	fileMutex sync.Mutex
)
var nodeName string

type Content struct {
	x string
}

type ConsensusState struct {
	currentBlockID         int
	receivedMinLeaderValue int
	ownLeaderValue         int
}

// Initialize the state
var state ConsensusState

var (
	config Config
	node   host.Host
)

type Message struct {
	Time    time.Time `json:"time"`
	Content string    `json:"content"`
}

type JSONRPCServer struct {
	mux      sync.RWMutex
	messages []Message
}

type Block struct {
	id           int
	prev_id      int
	leader_value int
	messages     []string
}

var blockchain []Block

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

// Broadcast sends a message to all connected nodes.
func (s *JSONRPCServer) Broadcast(message string) (interface{}, *jsonrpc.RPCError) {
	// Get the current time
	time := time.Now()

	// Create a new message struct with the current time and the given message
	msg := Message{Time: time, Content: message}

	// Lock the mutex to ensure thread safety
	s.mux.Lock()
	// Append the new message to the slice of messages
	s.messages = append(s.messages, msg)
	// Unlock the mutex
	s.mux.Unlock()

	// Print the broadcast message to the console
	fmt.Println("Broadcasting:", msg)

	// Update the file with the new message
	SaveTransaction(msg)

	// Iterate through all connected peers and send the message to each one
	for _, peer := range config.Peers {
		//fmt.Println("Sending message to peer", peer)
		// Extract the node ID from the peer string
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		// Get the peer's public key from the node list
		peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
		// Create the peer address from the peer string and the peer ID
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		// Connect to the peer if not already connected
		ConnectToPeer(node, peerID, address)
		// Get the time parts from the current time string
		timeParts := strings.Split(time.String(), " ")
		// Create the timestamp string in the correct format
		timeStr := strings.TrimSpace(timeParts[0] + "T" + timeParts[1] + "Z")
		// Send the message to the peer using the peer address and the timestamp string
		SendMessage(node, address, timeStr+"|"+message+"\n", "/transactions")
	}

	// Return nil to indicate that the method executed successfully
	return nil, nil
}

// TODO: Update query all
func (s *JSONRPCServer) QueryAll() ([]string, *jsonrpc.RPCError) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var result []string

	filePath := "./data/sorted_messages.txt"
	lines, _ := readAllLines(filePath)

	for _, line := range lines {
		parts := strings.SplitN(line, "|", 2)
		result = append(result, parts[1])

	}

	return result, nil
}

// handleJSONRPC is a HTTP handler function that processes JSON-RPC requests.
// It supports two methods: "Node.Broadcast" and "Node.QueryAll".
func handleJSONRPC(s *JSONRPCServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JSONRPC string        `json:"jsonrpc"`
			Method  string        `json:"method"`
			Params  []interface{} `json:"params"`
			ID      interface{}   `json:"id"`
		}

		// Decode the JSON-RPC request from the request body.
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON request", http.StatusBadRequest)
			return
		}

		var result interface{}
		var rpcErr *jsonrpc.RPCError

		// Process the JSON-RPC request based on the method.
		switch req.Method {
		case "Node.Broadcast":
			// Check if the request contains at least one parameter.
			if len(req.Params) > 0 {
				// Extract the message from the first parameter.
				if message, ok := req.Params[0].(string); ok {
					// Call the Broadcast method of the JSONRPCServer struct.
					result, rpcErr = s.Broadcast(message)
				} else {
					// Return an error if the first parameter is not a string.
					rpcErr = &jsonrpc.RPCError{Code: -32602, Message: "Invalid params"}
				}
			}
		case "Node.QueryAll":
			// Call the QueryAll method of the JSONRPCServer struct.
			result, rpcErr = s.QueryAll()
		default:
			// Return an error if the method is not supported.
			rpcErr = &jsonrpc.RPCError{Code: -32601, Message: "Method not found"}
		}

		// Prepare the JSON-RPC response.
		response := struct {
			JSONRPC string            `json:"jsonrpc"`
			ID      interface{}       `json:"id"`
			Result  interface{}       `json:"result,omitempty"`
			Error   *jsonrpc.RPCError `json:"error,omitempty"`
		}{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
			Error:   rpcErr,
		}

		// Set the Content-Type header of the response to application/json.
		w.Header().Set("Content-Type", "application/json")
		// Encode the JSON-RPC response and write it to the response body.
		if err := json.NewEncoder(w).Encode(response); err != nil {
			fmt.Println("Error encoding response:", err)
		}
	}
}

func StartJSONRPCServer(port int, server *JSONRPCServer) {
	http.HandleFunc("/rpc", handleJSONRPC(server))
	fmt.Printf("Starting JSON-RPC server on all interfaces, port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", port), nil); err != nil {
		fmt.Println("Error starting JSON-RPC server:", err)
	}
}

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
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
		peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		ConnectToPeer(node, peerID, address)
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

// SaveTransaction updates the file with a new message and sends it to all peers.
// It checks if the message already exists in the file before appending it.
// If the message is unique, it sorts the lines based on the timestamp and writes them back to the file.
func SaveTransaction(message Message) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Read the current contents of the file
	filePath := "./data/sorted_messages.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}

	// Format the new message in the correct format
	newLine := fmt.Sprintf("%s|%s", message.Time.Format(time.RFC3339), message.Content)

	// Check if the message already exists
	for _, line := range lines {
		if line == newLine {
			//fmt.Println("Duplicate message found, skipping:", newLine)
			return nil
		}
	}

	// Send the new message to all peers
	for _, peer := range config.Peers {
		//fmt.Println("Sending message to peer " + peer)
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		ConnectToPeer(node, peerID, address)
		SendMessage(node, address, newLine+"\n", "/transactions")
	}

	// Append the new message if it's unique
	lines = append(lines, newLine)

	// Sort the lines based on the timestamp
	sort.SliceStable(lines, func(i, j int) bool {
		time1, _ := time.Parse(time.RFC3339, strings.SplitN(lines[i], "|", 2)[0])
		time2, _ := time.Parse(time.RFC3339, strings.SplitN(lines[j], "|", 2)[0])
		return time1.Before(time2)
	})

	// Write the sorted lines back to the file
	return writeAllLines(filePath, lines)
}

func removeTransaction(targetLine string) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	filePath := "./data/sorted_messages.txt"
	targetLine = strings.TrimSpace(targetLine)
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}

	var newLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, targetLine) {
			newLines = append(newLines, line)
		}
	}

	err = writeAllLines(filePath, newLines)
	if err != nil {
		fmt.Println("Error writing updated transactions:", err)
		return err
	}

	return nil
}

// readAllLines reads all lines from the file
func readAllLines(filePath string) ([]string, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func writeAllLines(filePath string, lines []string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}

func SendMessage(h host.Host, peerAddr, message string, protocol prt.ID) {
	ctx := context.Background()
	peerMultiAddr, err := ma.NewMultiaddr(peerAddr)
	if err != nil {
		fmt.Println("Error parsing multiaddress:", err)
		return
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(peerMultiAddr)
	if err != nil {
		fmt.Println("Error converting to peer info:", err)
		return
	}

	stream, err := h.NewStream(ctx, peerInfo.ID, protocol)
	if err != nil {
		fmt.Println("Error opening stream to peer:", peerInfo.ID, err)
		return
	}
	defer stream.Close()

	_, err = stream.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending message to peer:", peerInfo.ID, err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the path to the configuration file as an argument.")
		return
	}
	configPath := os.Args[1]

	blockchainFilePath := "./data/blockchain.txt"
	blockchainLines, err := readAllLines(blockchainFilePath)
	if len(blockchainLines) > 0 {
		lastLine := blockchainLines[len(blockchainLines)-1]
		parts := strings.SplitN(lastLine, "/", 4)
		if len(parts) >= 1 {
			state.currentBlockID, _ = strconv.Atoi(parts[0])
		}
	}

	server := &JSONRPCServer{}
	go StartJSONRPCServer(7654, server)

	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	nodeName = "node" + numbers

	err = DecodeConfig(configPath, config)
	if err != nil {
		fmt.Println("Error decoding config:", err)
		return
	}
	
	fmt.Println(config.PrivateKey)
	node, err := RetrieveNodeFromPrivateKey(config.PrivateKey)
	if err != nil {
		fmt.Println("Error calculating peer ID:", err)
		return
	}
	fmt.Println("Node Addresses:", node.Addrs())

	// Open incoming peerstream
	node.SetStreamHandler("/peers", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		//fmt.Println("Received message:" + receivedString)

		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}

		if receivedString != nodeName && !contains(config.Peers, receivedString) {
			config.Peers = append(config.Peers, receivedString)

		}

		for _, peer := range config.Peers {
			//fmt.Println("Sending message to peer " + peer)
			re := regexp.MustCompile(`\d+`)
			id, _ := strconv.Atoi(re.FindString(peer))
			peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
			ConnectToPeer(node, peerID, address)

			filePath := "./data/blockchain.txt"
			lines, _ := readAllLines(filePath)

			// Check if the message already exists
			for _, line := range lines {
				SendMessage(node, address, line+"\n", "/blockchain")
			}
		}
	})
	// Open incoming transactions
	node.SetStreamHandler("/transactions", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		//fmt.Println("Received message:" + receivedString)
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}

		parts := strings.SplitN(receivedString, "|", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid input format:", receivedString)
			return
		}
		content := strings.TrimSpace(parts[1])
		time, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			fmt.Println("Error parsing time:", err)
			return
		}

		// Generate new message
		message := Message{
			Time:    time,
			Content: content,
		}

		// Update file
		err = SaveTransaction(message)
		if err != nil {
			fmt.Println("Error updating file:", err)
			return
		}
	})

	node.SetStreamHandler("/consensus", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		fmt.Println("Received Consensus Message:" + receivedString)
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}

		parts := strings.SplitN(receivedString, "|", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid input format:", receivedString)
			return
		}
		currentBlockID, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		if currentBlockID < state.currentBlockID {
			fmt.Println("Old ID received:", currentBlockID)
			return
		}
		if currentBlockID > state.currentBlockID {
			fmt.Println("Newer ID received:", currentBlockID)
			for _, peer := range config.Peers {
				re := regexp.MustCompile(`\d+`)
				id, _ := strconv.Atoi(re.FindString(peer))
				peerID, err := GetPeerIDFromPublicKey(config.Miners[id-1])

				if err != nil {
					fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
					continue
				}
				address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
				SendMessage(node, address, nodeName+"\n", "/peers")
			}
			return
		}

		leaderValue, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			fmt.Println("Error converting string:", err)
			return
		}

		// Update state
		state.currentBlockID = currentBlockID
		if state.receivedMinLeaderValue > leaderValue {
			state.receivedMinLeaderValue = leaderValue
		}
	})
	// Open incoming transactions
	node.SetStreamHandler("/blockchain", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		//fmt.Println("Received Block Message:" + receivedString)
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
		// var filePath = "./data/sorted_messages.txt"
		// var blockTransactions []stringm
		// lines, _ := readAllLines(filePath)
		// for i := 0; i < config.MinedBlockSize; i++ {
		// 	blockTransactions = append(blockTransactions, lines[i])
		// }
		// var root, _ = MerkleRootHash(blockTransactions)
		// state.ownLeaderValue = CalculateLeaderValue(root)
	})

	time.Sleep(1 * time.Second)

	for _, peer := range config.Peers {

		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, err := GetPeerIDFromPublicKey(config.Miners[id-1])

		if err != nil {
			fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
			continue
		}
		//fmt.Printf("Trying to connect to node %s with public key %s (Peer ID: %s)\n", config.Peers[i], miner, peerID)
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)

		if err := ConnectToPeer(node, peerID, address); err != nil {
			fmt.Printf("Error connecting to peer %s: %v\n", peerID, err)
		}

		for _, knownPeer := range config.Peers {

			if knownPeer != peer {
				SendMessage(node, address, knownPeer+"\n", "/peers")
			}
		}
		SendMessage(node, address, nodeName+"\n", "/peers")
	}

	time.Sleep(1 * time.Second)

	for _, peer := range config.Peers {

		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, err := GetPeerIDFromPublicKey(config.Miners[id-1])

		if err != nil {
			fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
			continue
		}

		//fmt.Printf("Trying to connect to node %s with public key %s (Peer ID: %s)\n", config.Peers[i], miner, peerID)
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)

		if err := ConnectToPeer(node, peerID, address); err != nil {
			fmt.Printf("Error connecting to peer %s: %v\n", peerID, err)
		}

		for _, knownPeer := range config.Peers {

			if knownPeer != peer {
				SendMessage(node, address, knownPeer+"\n", "/peers")
			}
		}
		SendMessage(node, address, nodeName+"\n", "/peers")
	}
	// time.Sleep(1 * time.Second)
	// message := Message{
	// 	Time:    time.Now(),
	// 	Content: "Message from " + nodeName,
	// }
	// SaveTransaction(message)
	// for _, peer := range config.Peers {
	// 	fmt.Println("Sending message to peer " + peer)
	// 	re := regexp.MustCompile(`\d+`)
	// 	id, _ := strconv.Atoi(re.FindString(peer))
	// 	peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
	// 	address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
	// 	ConnectToPeer(node, peerID, address)
	// 	timeParts := strings.Split(message.Time.String(), " ")
	// 	timeStr := strings.TrimSpace(timeParts[0] + "T" + timeParts[1] + "Z")
	// 	SendMessage(node, address, timeStr+"|Message from "+nodeName+"\n", "/transactions")
	// }

	time.Sleep(1 * time.Second)

	SaveConfig(&config, configPath)
	// time.Sleep(10 * time.Second)
	var startConsensus = time.Now()
	for {

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
				state.currentBlockID, _ = strconv.Atoi(parts[0])
				state.currentBlockID += 1
			}
		}

		var filePath = "./data/sorted_messages.txt"
		var BlockTransactions []string
		lines, _ := readAllLines(filePath)

		if len(lines) < config.MinedBlockSize {
			fmt.Println("Not enough transactions to create a block. Waiting for new transactions...")
			// time.Sleep(time.Second) // Sleep briefly before checking again
			continue
		}

		fmt.Println("Appending transactions")
		for i := 0; i < config.MinedBlockSize; i++ {
			BlockTransactions = append(BlockTransactions, lines[i])
		}
		root, _ := MerkleRootHash(BlockTransactions)
		state.ownLeaderValue = CalculateLeaderValue(root)
		state.receivedMinLeaderValue = state.ownLeaderValue
		fmt.Println("Sending Leadervalue")
		for _, peer := range config.Peers {
			re := regexp.MustCompile(`\d+`)
			id, _ := strconv.Atoi(re.FindString(peer))
			peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
			SendMessage(node, address, fmt.Sprintf("%d|%d", state.currentBlockID, state.ownLeaderValue)+"\n", "/consensus")
		}

		// Wait for the consensus timeout duration
		// time.Sleep(10 * time.Second)
		nextMinute := startConsensus.Truncate(time.Minute).Add(time.Minute * 2)
		var endConsensus = nextMinute
		for {
			if time.Now().After(endConsensus) {
				break
			}
		}
		// time.Sleep(endConsensus.Sub(time.Now()))
		startConsensus = time.Now()
		fmt.Println(startConsensus, endConsensus)

		if state.receivedMinLeaderValue >= state.ownLeaderValue {
			var newBlock Block
			newBlock.id = state.currentBlockID
			newBlock.prev_id = state.currentBlockID - 1
			newBlock.leader_value = state.ownLeaderValue

			fmt.Println("Creating new block")
			for i := 0; i < config.MinedBlockSize; i++ {
				newBlock.messages = append(newBlock.messages, lines[i])
			}
			fmt.Println(newBlock)
			state.currentBlockID, _ = SaveBlock(newBlock)
			lines, _ = readAllLines("./data/blockchain.txt")
			for _, peer := range config.Peers {
				re := regexp.MustCompile(`\d+`)
				id, _ := strconv.Atoi(re.FindString(peer))
				peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
				address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
				SendMessage(node, address, lines[len(lines)-1], "/blockchain")
				fmt.Println("Sending Block")
			}
			fmt.Println("Ended loop iteration")
		}
	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
