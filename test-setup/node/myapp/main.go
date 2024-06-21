package main

import (
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	ethState "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	nodeName string
	config   Config
	node     host.Host
	state    ConsensusState
	evm      *vm.EVM
	stateDB  *ethState.StateDB
	blockCtx vm.BlockContext
	txCtx    vm.TxContext
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

	//Start EVM

	// Initialize the state database (persistent)
	db := rawdb.NewMemoryDatabase()
	rootHash := common.Hash{}
	stateDB, _ = ethState.New(rootHash, ethState.NewDatabase(db), nil)

	// Initialize genesis accounts and balances
	for addr, balance := range config.Addresses {
		slicedHex := addr[2:]
		address := common.HexToAddress(slicedHex)
		stateDB.CreateAccount(address)
		stateDB.SetBalance(address, uint256.MustFromBig(big.NewInt(int64(balance))), 0)
	}

	blockCtx = vm.BlockContext{
		Coinbase:    common.HexToAddress("0x0000000000000000000000000000000000000000"), // Default coinbase address
		GasLimit:    10000000,                                                          // Default gas limit
		BlockNumber: big.NewInt(0),
	}

	txCtx = vm.TxContext{
		Origin:   common.HexToAddress("0x0000000000000000000000000000000000000000"), // Default origin address
		GasPrice: big.NewInt(0),                                                     // Assuming no gas price for simplicity
	}
	evm = initializeEVM(blockCtx, txCtx, stateDB)

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
