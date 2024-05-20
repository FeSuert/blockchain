package main

import (
	"bufio"
	_ "context"
	_ "crypto/sha256"
	_ "encoding/json"
	"fmt"
	_ "net/http"
	"os"
	"os/signal"
	"regexp"
	_ "sort"
	"strconv"
	"strings"
	_ "sync"
	"syscall"
	"time"

	_ "github.com/cbergoon/merkletree"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	_ "github.com/libp2p/go-libp2p/core/peer"
	_ "github.com/ybbus/jsonrpc"
)

var nodeName string

// Initialize the state
var state ConsensusState

var (
	config Config
	node   host.Host
)

var blockchain []Block

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the path to the configuration file as an argument.")
		return
	}
	configPath := os.Args[1]

	blockchainFilePath := "./data/blockchain.txt"
	blockchainLines, err := ReadAllLines(blockchainFilePath)
	if len(blockchainLines) > 0 {
		lastLine := blockchainLines[len(blockchainLines)-1]
		parts := strings.SplitN(lastLine, "/", 4)
		if len(parts) >= 1 {
			state.currentBlockID, _ = strconv.Atoi(parts[0])
		}
	}

	server := &JSONRPCServer{}
	go StartJSONRPCServer(7654, server, config, node)

	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	nodeName = "node" + numbers

	config, err = DecodeConfig(configPath, config)
	if err != nil {
		fmt.Println("Error decoding config:", err)
		return
	}
	node, err := RetrieveNodeFromPrivateKey(config.PrivateKey, nodeName)
	if err != nil {
		fmt.Println("Error calculating peer ID:", err)
		return
	}
	fmt.Println("Node Addresses:", node.Addrs())

	// Open incoming peerstream
	OpenPeerStream(node, &config, nodeName)

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
		err = SaveTransaction(message, config, node)
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
		state.currentBlockID, err = SaveBlock(receivedBlock, state, config, node)
		if err != nil {
			fmt.Println("Error updating file:", err)
			return
		}
		// var filePath = "./data/sorted_messages.txt"
		// var blockTransactions []stringm
		// lines, _ := ReadAllLines(filePath)
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
	//var startConsensus = time.Now()
	// for {

	// 	blockchainFilePath := "./data/blockchain.txt"
	// 	blockchainLines, err := ReadAllLines(blockchainFilePath)
	// 	if err != nil {
	// 		fmt.Println("Error reading blockchain file:", err)
	// 		continue
	// 	}
	// 	if len(blockchainLines) > 0 {
	// 		lastLine := blockchainLines[len(blockchainLines)-1]
	// 		parts := strings.SplitN(lastLine, "/", 4)
	// 		if len(parts) >= 1 {
	// 			state.currentBlockID, _ = strconv.Atoi(parts[0])
	// 			state.currentBlockID += 1
	// 		}
	// 	}

	// 	var filePath = "./data/sorted_messages.txt"
	// 	var BlockTransactions []string
	// 	lines, _ := ReadAllLines(filePath)

	// 	if len(lines) < config.MinedBlockSize {
	// 		fmt.Println("Not enough transactions to create a block. Waiting for new transactions...")
	// 		// time.Sleep(time.Second) // Sleep briefly before checking again
	// 		continue
	// 	}

	// 	fmt.Println("Appending transactions")
	// 	for i := 0; i < config.MinedBlockSize; i++ {
	// 		BlockTransactions = append(BlockTransactions, lines[i])
	// 	}
	// 	root, _ := MerkleRootHash(BlockTransactions)
	// 	state.ownLeaderValue = CalculateLeaderValue(root)
	// 	state.receivedMinLeaderValue = state.ownLeaderValue
	// 	fmt.Println("Sending Leadervalue")
	// 	for _, peer := range config.Peers {
	// 		re := regexp.MustCompile(`\d+`)
	// 		id, _ := strconv.Atoi(re.FindString(peer))
	// 		peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
	// 		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
	// 		SendMessage(node, address, fmt.Sprintf("%d|%d", state.currentBlockID, state.ownLeaderValue)+"\n", "/consensus")
	// 	}

	// 	// Wait for the consensus timeout duration
	// 	// time.Sleep(10 * time.Second)
	// 	nextMinute := startConsensus.Truncate(time.Minute).Add(time.Minute * 2)
	// 	var endConsensus = nextMinute
	// 	for {
	// 		if time.Now().After(endConsensus) {
	// 			break
	// 		}
	// 	}
	// 	// time.Sleep(endConsensus.Sub(time.Now()))
	// 	startConsensus = time.Now()
	// 	fmt.Println(startConsensus, endConsensus)

	// 	if state.receivedMinLeaderValue >= state.ownLeaderValue {
	// 		var newBlock Block
	// 		newBlock.id = state.currentBlockID
	// 		newBlock.prev_id = state.currentBlockID - 1
	// 		newBlock.leader_value = state.ownLeaderValue

	// 		fmt.Println("Creating new block")
	// 		for i := 0; i < config.MinedBlockSize; i++ {
	// 			newBlock.messages = append(newBlock.messages, lines[i])
	// 		}
	// 		fmt.Println(newBlock)
	// 		state.currentBlockID, _ = SaveBlock(newBlock)
	// 		lines, _ = ReadAllLines("./data/blockchain.txt")
	// 		for _, peer := range config.Peers {
	// 			re := regexp.MustCompile(`\d+`)
	// 			id, _ := strconv.Atoi(re.FindString(peer))
	// 			peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
	// 			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
	// 			SendMessage(node, address, lines[len(lines)-1], "/blockchain")
	// 			fmt.Println("Sending Block")
	// 		}
	// 		fmt.Println("Ended loop iteration")
	// 	}
	// }

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
