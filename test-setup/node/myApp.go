package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
	prt "github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Peers          []string `toml:"peers"`
	RPCPort        int64    `toml:"rpc_port"`
	SendPort       int64    `toml:"send_port"`
	Miners         []string `toml:"miners"`
	PrivateKey     string   `toml:"private_key"`
	MinedBlockSize int64    `toml:"mined_block_size"`
}

type IDs map[string]string

var Nodename string

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

func connectToPeer(h host.Host, ctx context.Context, hostAddr string, peerAddr string) (*peerstore.AddrInfo, error) {
	hostMultiAddr, err := ma.NewMultiaddr(hostAddr)
	if err != nil {
		return nil, err
	}
	peerMultiAddr, err := ma.NewMultiaddr(peerAddr)
	if err != nil {
		return nil, err
	}

	peerID, err := peer.AddrInfoFromP2pAddr(peerMultiAddr)
	if err != nil {
		return nil, err
	}

	peerID.Addrs = append(peerID.Addrs, hostMultiAddr)

	if err := h.Connect(ctx, *peerID); err != nil {
		return nil, err
	}

	fmt.Println("Connected to Peer:", peerID.ID)
	return peerID, nil
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

func handleStream(s network.Stream) {
	reader := bufio.NewReader(s)
	message, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading from stream:", err)
		s.Close()
		return
	}

	fmt.Println("Received message:", message)

	ackMsg := "ACK\n"
	_, err = s.Write([]byte(ackMsg))
	if err != nil {
		fmt.Println("Failed to send ACK:", err)
	}
	s.Close()
}

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the path to the configuration file as an argument.")
		return
	}
	configPath := os.Args[1]

	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	Nodename = "node" + numbers

	var config Config

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

	fmt.Println("Peers:", config.Peers)
	fmt.Println("RPC Port:", config.RPCPort)
	fmt.Println("Send Port:", config.SendPort)

	node, err := libp2p.New(libp2p.ListenAddrStrings(fmt.Sprintf("/dns4/%s/tcp/8443", Nodename)))
	if err != nil {
		panic(err)
	}
	fmt.Println(node.Addrs())
	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		fmt.Println("Error getting node address:", err)
		return
	}
	fmt.Println("libp2p node address:", addrs[0])

	filePath := "ids.toml"
	num, err := strconv.Atoi(numbers)
	if err != nil {
		fmt.Println("Invalid node number:", err)
		return
	}
	time.Sleep(time.Duration(num+3) * time.Second)
	tree, err := toml.LoadFile(filePath)
	if err != nil {
		tree, err = toml.TreeFromMap(map[string]interface{}{})
		if err != nil {
			fmt.Println("Error initializing TOML tree:", err)
			return
		}
	}

	tree.Set(Nodename, addrs[0].String())

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error opening file for writing:", err)
		return
	}
	defer f.Close()
	if _, err := tree.WriteTo(f); err != nil {
		fmt.Println("Error writing TOML data to file:", err)
		return
	}

	time.Sleep(5 * time.Second)

	file, err = os.Open("ids.toml")
	if err != nil {
		log.Fatal("Error opening file:", err)
	}
	defer file.Close()

	var configToml IDs
	if err := toml.NewDecoder(file).Decode(&configToml); err != nil {
		fmt.Println("Error reading config file:", err)
		return
	}

	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}

		fmt.Printf("Address for %s is %s\n", peer, nodeAddress)
		ctx := context.Background()
		_, err = connectToPeer(node, ctx, node.Addrs()[0].String(), nodeAddress)
		if err != nil {
			fmt.Println("Error connecting to peer:", err)
		}
	}

	node.SetStreamHandler("/chat/1.0.0", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}
		fmt.Println("Received string:", receivedString)
	})

	node.SetStreamHandler("/application/1.0.0", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}
		fmt.Println("Received string:", receivedString)
		if receivedString != Nodename && !contains(config.Peers, receivedString) {
			config.Peers = append(config.Peers, receivedString)
			fmt.Println("Updated Peers:", config.Peers)
		}
	})

	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}

		fmt.Printf("Address for %s is %s\n", peer, nodeAddress)
		ctx := context.Background()
		_, err = connectToPeer(node, ctx, node.Addrs()[0].String(), nodeAddress)
		if err != nil {
			fmt.Println("Error connecting to peer:", err)
		}
	}

	time.Sleep(10 * time.Second)
	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}
		for _, peer1 := range config.Peers {
			SendMessage(node, nodeAddress, peer1+"\n", "/application/1.0.0")
			SendMessage(node, nodeAddress, Nodename+"\n", "/application/1.0.0")
		}
	}
	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}
		SendMessage(node, nodeAddress, "Hello from "+Nodename+"\n", "/chat/1.0.0")
	}
	time.Sleep(5 * time.Second)

	saveConfig(&config, configPath)

	time.Sleep(10 * time.Second)
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
