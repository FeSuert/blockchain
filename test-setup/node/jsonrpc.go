package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ybbus/jsonrpc"
)

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
	SaveTransaction(msg)

	// Iterate through all connected peers and send the message to each one
	for _, peer := range config.Peers {
		//fmt.Println("Sending message to peer", peer)
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
