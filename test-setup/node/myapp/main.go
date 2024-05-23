package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	nodeName string
	config   Config
	node     host.Host
	state    ConsensusState
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the path to the configuration file as an argument.")
		return
	}
	configPath := os.Args[1]

	server := &JSONRPCServer{}
	go StartJSONRPCServer(7654, server)

	nodeName = extractNodeName(configPath)

	if err := loadConfig(configPath, &config); err != nil {
		fmt.Println("Error loading config file:", err)
		return
	}

	priv, err := createPrivateKey(config.PrivateKey)
	if err != nil {
		panic(err)
	}

	node, err = createLibp2pHost(priv, nodeName)
	if err != nil {
		panic(err)
	}

	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		panic(err)
	}
	fmt.Println("Node Peer ID:", peerID)

	node, err = createLibp2pHost(priv, nodeName)
	if err != nil {
		panic(err)
	}

	currentBlockID, err := GetCurrentBlockID()
	if err != nil {
		fmt.Println("Error getting current block ID:", err)
		return
	}

	state := &ConsensusState{
		CurrentBlockID:         currentBlockID,
		ReceivedMinLeaderValue: int(^uint(0) >> 1), // Initialize to max int
		OwnLeaderValue:         0,
	}

	initializeStreamHandlers(node, state)
	time.Sleep(1 * time.Second)
	connectToPeers(node, config.Peers, config.Miners)

	time.Sleep(1 * time.Second)

	go startConsensus(node, config, state)

	time.Sleep(1 * time.Second)

	saveConfig(&config, configPath)

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
