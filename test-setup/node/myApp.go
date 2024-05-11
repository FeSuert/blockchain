package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	prt "github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml"
	"github.com/ybbus/jsonrpc"
)

var (
	fileMutex sync.Mutex
)

type Config struct {
	Peers          []string `toml:"peers"`
	RPCPort        int64    `toml:"rpc_port"`
	SendPort       int64    `toml:"send_port"`
	Miners         []string `toml:"miners"`
	PrivateKey     string   `toml:"private_key"`
	MinedBlockSize int64    `toml:"mined_block_size"`
}

var nodeName string

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
	UpdateFile(msg)

	// Iterate through all connected peers and send the message to each one
	for _, peer := range config.Peers {
		fmt.Println("Sending message to peer", peer)
		// Extract the node ID from the peer string
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		// Get the peer's public key from the node list
		peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
		// Create the peer address from the peer string and the peer ID
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		// Connect to the peer if not already connected
		connectToPeer(node, peerID, address)
		// Get the time parts from the current time string
		timeParts := strings.Split(time.String(), " ")
		// Create the timestamp string in the correct format
		timeStr := strings.TrimSpace(timeParts[0] + "T" + timeParts[1] + "Z")
		// Send the message to the peer using the peer address and the timestamp string
		SendMessage(node, address, timeStr+"|"+message+"\n", "/chat")
	}

	// Return nil to indicate that the method executed successfully
	return nil, nil
}

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

// UpdateFile updates the file with a new message and sends it to all peers.
// It checks if the message already exists in the file before appending it.
// If the message is unique, it sorts the lines based on the timestamp and writes them back to the file.
func UpdateFile(message Message) error {
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
			fmt.Println("Duplicate message found, skipping:", newLine)
			return nil
		}
	}

	// Send the new message to all peers
	for _, peer := range config.Peers {
		fmt.Println("Sending message to peer " + peer)
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		connectToPeer(node, peerID, address)
		SendMessage(node, address, newLine+"\n", "/chat")
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

func saveConfig(config *Config, configPath string) {
	data, err := toml.Marshal(config)
	if err != nil {
		log.Fatalf("Error marshalling config: %s", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		log.Fatalf("Error writing config to file: %s", err)
	}
}

func connectToPeer(h host.Host, peerID peer.ID, address string) error {
	ctx := context.Background()
	peerAddr, err := ma.NewMultiaddr(address)
	if err != nil {
		return fmt.Errorf("error creating multiaddr: %v", err)
	}
	peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
	if err != nil {
		return fmt.Errorf("error creating peer info: %v", err)
	}

	if err := h.Connect(ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", peerInfo.ID, err)
	}
	fmt.Printf("Successfully connected to peer %s\n", peerInfo.ID)
	return nil
}

func getPeerIDFromPublicKey(pubKeyHex string) (peer.ID, error) {
	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("error decoding public key: %v", err)
	}
	libp2pPubKey, err := crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling public key: %v", err)
	}
	return peer.IDFromPublicKey(libp2pPubKey)
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

	server := &JSONRPCServer{}
	go StartJSONRPCServer(7654, server)

	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	nodeName = "node" + numbers

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Println("Error opening config file:", err)
		return
	}
	defer file.Close()

	if err := toml.NewDecoder(file).Decode(&config); err != nil {
		fmt.Println("Error reading config file:", err)
		return
	}

	// Convert hex string to bytes
	privateKeyHex := strings.TrimLeft(config.PrivateKey, "0x")
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		panic(err)
	}

	// Create Ed25519 private key from seed
	priv := ed25519.NewKeyFromSeed(privateKeyBytes)

	// Convert private key to libp2p format
	libp2pPrivKey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	if err != nil {
		panic(err)
	}
	// Calculate the peer ID using the private key
	peerID, err := peer.IDFromPrivateKey(libp2pPrivKey)
	if err != nil {
		panic(err)
	}
	fmt.Println("Node Peer ID:", peerID)
	// Generate new node
	node, err = libp2p.New(
		libp2p.Identity(libp2pPrivKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/dns4/%s/tcp/8080", nodeName)),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Node Addresses:", node.Addrs())

	// Open incoming peerstream
	node.SetStreamHandler("/peers", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		fmt.Println("Received message:" + receivedString)

		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}

		if receivedString != nodeName && !contains(config.Peers, receivedString) {
			config.Peers = append(config.Peers, receivedString)

		}

		for _, peer := range config.Peers {
			fmt.Println("Sending message to peer " + peer)
			re := regexp.MustCompile(`\d+`)
			id, _ := strconv.Atoi(re.FindString(peer))
			peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
			connectToPeer(node, peerID, address)

			filePath := "./data/sorted_messages.txt"
			lines, _ := readAllLines(filePath)

			// Check if the message already exists
			for _, line := range lines {
				SendMessage(node, address, line+"\n", "/chat")
			}
		}
	})

	// Open incoming chatstream
	node.SetStreamHandler("/chat", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		fmt.Println("Received message:" + receivedString)
		if err != nil {
			fmt.Println("Fehler beim Lesen des eingehenden Strings:", err)
		}

		parts := strings.SplitN(receivedString, "|", 2)
		if len(parts) != 2 {
			fmt.Println("Ungültiges Eingabeformat:", receivedString)
			return
		}
		content := strings.TrimSpace(parts[1])
		time, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			fmt.Println("Fehler beim Parsen der Zeit:", err)
			return
		}

		// Generate new message
		message := Message{
			Time:    time,
			Content: content,
		}

		// Update file
		err = UpdateFile(message)
		if err != nil {
			fmt.Println("Fehler beim Aktualisieren der Datei:", err)
			return
		}
	})

	time.Sleep(1 * time.Second)

	for _, peer := range config.Peers {

		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, err := getPeerIDFromPublicKey(config.Miners[id-1])

		if err != nil {
			fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
			continue
		}
		//fmt.Printf("Trying to connect to node %s with public key %s (Peer ID: %s)\n", config.Peers[i], miner, peerID)
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)

		if err := connectToPeer(node, peerID, address); err != nil {
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
		peerID, err := getPeerIDFromPublicKey(config.Miners[id-1])

		if err != nil {
			fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
			continue
		}

		//fmt.Printf("Trying to connect to node %s with public key %s (Peer ID: %s)\n", config.Peers[i], miner, peerID)
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)

		if err := connectToPeer(node, peerID, address); err != nil {
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
	// UpdateFile(message)
	// for _, peer := range config.Peers {
	// 	fmt.Println("Sending message to peer " + peer)
	// 	re := regexp.MustCompile(`\d+`)
	// 	id, _ := strconv.Atoi(re.FindString(peer))
	// 	peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
	// 	address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
	// 	connectToPeer(node, peerID, address)
	// 	timeParts := strings.Split(message.Time.String(), " ")
	// 	timeStr := strings.TrimSpace(timeParts[0] + "T" + timeParts[1] + "Z")
	// 	SendMessage(node, address, timeStr+"|Message from "+nodeName+"\n", "/chat")
	// }

	time.Sleep(1 * time.Second)

	saveConfig(&config, configPath)

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
