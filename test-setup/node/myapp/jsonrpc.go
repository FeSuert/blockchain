package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ybbus/jsonrpc"
)

type JSONRPCServer struct {
	mux      sync.RWMutex
	messages []Message
}

func (s *JSONRPCServer) Broadcast(message string) (interface{}, *jsonrpc.RPCError) {
	time := time.Now()
	tx, err := parseTransactionFromJSON(message)
	if err != nil {
		fmt.Println("Error parsing transaction JSONRPC:", err)
		return nil, &jsonrpc.RPCError{Code: -32602, Message: "Invalid transaction format"}
	}
	msg := Message{Time: time, Content: tx}
	fmt.Println("Sender after Parsing: " + tx.Sender)

	s.mux.Lock()
	s.messages = append(s.messages, msg)
	s.mux.Unlock()

	UpdateFile(msg)

	return nil, nil
}

func (s *JSONRPCServer) QueryAll() ([]string, *jsonrpc.RPCError) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	//var result []string

	filePath := "./data/blockchain.txt"
	lines, _ := readAllLines(filePath)

	// for _, line := range lines {
	// 	parts := strings.SplitN(line, "|", 2)
	// 	result = append(result, parts[1])
	// }

	return lines, nil
}

func handleJSONRPC(s *JSONRPCServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JSONRPC string        `json:"jsonrpc"`
			Method  string        `json:"method"`
			Params  []interface{} `json:"params"`
			ID      interface{}   `json:"id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON request", http.StatusBadRequest)
			return
		}

		var result interface{}
		var rpcErr *jsonrpc.RPCError

		switch req.Method {
		case "Node.Broadcast":
			fmt.Println(req.Params)
			if len(req.Params) > 0 {
				if paramMap, ok := req.Params[0].(map[string]interface{}); ok {
					if message, err := json.Marshal(paramMap); err == nil {
						result, rpcErr = s.Broadcast(string(message))
					} else {
						rpcErr = &jsonrpc.RPCError{Code: -32602, Message: "Invalid params"}
					}
				} else {
					rpcErr = &jsonrpc.RPCError{Code: -32602, Message: "Invalid params"}
				}
			}
		case "Node.QueryAll":
			result, rpcErr = s.QueryAll()
		default:
			rpcErr = &jsonrpc.RPCError{Code: -32601, Message: "Method not found"}
		}

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

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			fmt.Println("Error encoding response:", err)
		}
	}
}

func StartJSONRPCServer(port int, server *JSONRPCServer) {
	http.HandleFunc("/rpc", handleJSONRPC(server))
	fmt.Printf("Starting JSON-RPC server on all interfaces, port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil); err != nil {
		fmt.Println("Error starting JSON-RPC server:", err)
	}
}
