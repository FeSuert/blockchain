package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pelletier/go-toml"
)

var (
	fileMutex sync.Mutex
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
	blockchainLines, _ := readAllLines(blockchainFilePath)
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
	node.SetStreamHandler("/peers", handlePeersMessage)
	// Open incoming transactions
	node.SetStreamHandler("/transactions", handleTransactionsMessage)
	node.SetStreamHandler("/consensus", handleConsensusMessage)
	node.SetStreamHandler("/blockchain", handleBlockchainMessage)

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

	saveConfig(&config, configPath)
	// time.Sleep(10 * time.Second)
	// for {

	// 	blockchainFilePath := "./data/blockchain.txt"
	// 	blockchainLines, err := readAllLines(blockchainFilePath)
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
	// 	lines, _ := readAllLines(filePath)

	// 	if len(lines) < config.MinedBlockSize {
	// 		fmt.Println("Not enough transactions to create a block. Waiting for new transactions...")
	// 		time.Sleep(time.Second) // Sleep briefly before checking again
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
	// 		peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
	// 		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
	// 		SendMessage(node, address, fmt.Sprintf("%d|%d", state.currentBlockID, state.ownLeaderValue)+"\n", "/consensus")
	// 	}

	// 	// Wait for the consensus timeout duration
	// 	time.Sleep(10 * time.Second)

	// 	if state.receivedMinLeaderValue >= state.ownLeaderValue {
	// 		var newBlock Block
	// 		newBlock.id = state.currentBlockID
	// 		newBlock.prev_id = state.currentBlockID - 1
	// 		newBlock.leader_value = state.ownLeaderValue

	// 		fmt.Println("Creating new block")
	// 		for i := 0; i < config.MinedBlockSize; i++ {
	// 			newBlock.messages = append(newBlock.messages, lines[i])
	// 		}
	// 		state.currentBlockID, _ = SaveBlock(newBlock)
	// 		lines, _ = readAllLines("./data/blockchain.txt")
	// 		for _, peer := range config.Peers {
	// 			re := regexp.MustCompile(`\d+`)
	// 			id, _ := strconv.Atoi(re.FindString(peer))
	// 			peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
	// 			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
	// 			SendMessage(node, address, lines[len(lines)-1], "/blockchain")
	// 		}
	// 	}
	// }

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
