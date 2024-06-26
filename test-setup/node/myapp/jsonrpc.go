package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/ybbus/jsonrpc"
)

type JSONRPCServer struct {
	mux      sync.RWMutex
	messages []Message
}

func (s *JSONRPCServer) TxResult(txHash string) (map[int]string, *jsonrpc.RPCError) {
	lines, err := readAllLines("./data/tx_results.txt")
	if err != nil {
		return nil, &jsonrpc.RPCError{Code: -32602, Message: "Invalid readLine call"}
	}
	var txH, result string
	for _, line := range lines {
		parts := strings.Split(line, "|")

		// Ensure the parts are as expected
		if len(parts) != 2 {
			fmt.Println("Error: expected 2 parts separated by '|'")
			return nil, &jsonrpc.RPCError{Code: -32602, Message: "Invalid tx result format"}
		}

		// Assign the split parts to the variables
		txH = parts[0]
		result = parts[1]

		if txH == txHash {
			// Create a map to hold the result in the required format
			resultMap := map[int]string{
				0: result,
			}
			return resultMap, nil
		}
	}

	return nil, &jsonrpc.RPCError{Code: -32602, Message: "No such transaction"}
}

func (s *JSONRPCServer) Broadcast(message string) (interface{}, *jsonrpc.RPCError) {
	time := time.Now()
	tx, err := parseTransactionFromJSON(message)
	if err != nil {
		fmt.Println("Error parsing transaction JSONRPC:", err)
		return nil, &jsonrpc.RPCError{Code: -32602, Message: "Invalid transaction format"}
	}
	msg := Message{Time: time, Content: tx}
	//fmt.Println("Sender after Parsing: " + tx.Sender)

	s.mux.Lock()
	s.messages = append(s.messages, msg)
	s.mux.Unlock()

	UpdateSortedMessages(msg)

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

func (s *JSONRPCServer) HighestBlock() (interface{}, *jsonrpc.RPCError) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return state.CurrentBlockID, nil
}

func (s *JSONRPCServer) QueryAddress(params []interface{}) (interface{}, *jsonrpc.RPCError) {
	if len(params) != 1 {
		return nil, &jsonrpc.RPCError{Code: -32602, Message: "Invalid params"}
	}
	paramMap, ok := params[0].(map[string]interface{})
	if !ok {
		return nil, &jsonrpc.RPCError{Code: -32602, Message: "Invalid param format"}
	}

	to, toOk := paramMap["to"].(string)
	input, inputOk := paramMap["input"].(string)
	sender, senderOk := paramMap["sender"].(string)
	if !toOk || !senderOk {
		return nil, &jsonrpc.RPCError{Code: -32602, Message: "Fields 'to' and 'sender' are required"}
	}

	// Handle balance query if input is not provided
	if !inputOk {
		address := common.HexToAddress(to)
		balance := evm.StateDB.GetBalance(address)
		return balance.String(), nil
	}

	// Handle smart contract call
	caller := vm.AccountRef(common.HexToAddress(sender))
	toAddr := common.HexToAddress(to)
	data := common.FromHex(input)

	result, _, err := evm.Call(caller, toAddr, data, uint64(1000000), new(uint256.Int))
	if err != nil {
		return nil, &jsonrpc.RPCError{Code: -32000, Message: fmt.Sprintf("EVM call error: %v", err)}
	}
	return common.Bytes2Hex(result), nil
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
			// fmt.Println(req.Params)
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
		case "Node.HighestBlock":
			result, rpcErr = s.HighestBlock()
		case "Node.QueryAddress":
			result, rpcErr = s.QueryAddress(req.Params)
		case "Node.TxResult":
			result, rpcErr = s.TxResult(req.Params[0].(string))
			// fmt.Println("Req.Params:", req.Params)
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
		//fmt.Println("Response:", response)
	}
}

func StartJSONRPCServer(port int, server *JSONRPCServer) {
	http.HandleFunc("/rpc", handleJSONRPC(server))
	fmt.Printf("Starting JSON-RPC server on all interfaces, port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil); err != nil {
		fmt.Println("Error starting JSON-RPC server:", err)
	}
}
