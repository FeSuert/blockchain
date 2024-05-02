package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
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

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json"
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
type Message struct {
	Time    time.Time
	Content string
}

var (
	fileMutex sync.Mutex
)

type IDs map[string]string

var Nodename string

type GossipService struct{}

type BroadcastRequest struct {
	Message string
}

type QueryAllResponse struct {
	Messages []string `json:"messages"`
}

var (
	config     Config
	configToml IDs
	node       host.Host
)

func (s *GossipService) Broadcast(r *http.Request, req *BroadcastRequest, res *struct{}) error {
	msg := Message{
		Time:    time.Now(),
		Content: req.Message,
	}
	fmt.Println("Broadcasting message" + req.Message)
	// Iterate through peers and send the message
	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}

		ctx := context.Background()
		_, err := connectToPeer(node, ctx, node.Addrs()[0].String(), nodeAddress)
		if err != nil {
			fmt.Println("Error connecting to peer:", err)
			continue
		}

		timeParts := strings.Split(msg.Time.String(), " ")
		timeStr := strings.TrimSpace(timeParts[0] + "T" + timeParts[1] + "Z")

		SendMessage(node, nodeAddress, timeStr+"|"+req.Message+"\n", "/chat/1.0.0")
	}

	return nil
}

func (s *GossipService) QueryAll(r *http.Request, req *struct{}, res *QueryAllResponse) error {
	// Reading messages from sorted_messages.txt
	filePath := "./data/sorted_messages.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}

	// Add messages to the response
	res.Messages = lines

	// Send messages to config.SendPort
	address := fmt.Sprintf("localhost:%d", config.SendPort)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting:", err)
		return err
	}
	defer conn.Close()

	for _, message := range lines {
		_, err = conn.Write([]byte(message + "\n"))
		if err != nil {
			fmt.Println("Error sending message:", err)
			return err
		}
	}

	return nil
}

func StartRPCServer() {
	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterService(new(GossipService), "")

	http.Handle("/rpc", s)

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", config.RPCPort), nil)
		if err != nil {
			fmt.Println("Error starting server:", err)
		}
	}()
}

func QueryAll(nodeName string, h host.Host, nodeAddress string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	filePath := "./data/sorted_messages.txt"
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	if nodeAddress == "" {
		fmt.Println("Address not found for node:", nodeName)
		return
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		message := scanner.Text()
		fmt.Println(message)
		SendMessage(h, nodeAddress, message, "/chat/1.0.0")
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
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
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the path to the configuration file as an argument.")
		return
	}
	configPath := os.Args[1]

	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	Nodename = "node" + numbers

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

	// fmt.Println("Peers:", config.Peers)
	// fmt.Println("RPC Port:", config.RPCPort)
	// fmt.Println("Send Port:", config.SendPort)

	node, err = libp2p.New(libp2p.ListenAddrStrings(fmt.Sprintf("/dns4/%s/tcp/8443", Nodename)))
	if err != nil {
		panic(err)
	}
	// fmt.Println(node.Addrs())
	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		fmt.Println("Error getting node address:", err)
		return
	}
	// fmt.Println("libp2p node address:", addrs[0])

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

	if err := toml.NewDecoder(file).Decode(&configToml); err != nil {
		fmt.Println("Error reading config file:", err)
		return
	}
	// fmt.Println(configToml)
	fmt.Println("Starting RPC Server")
	StartRPCServer()

	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}

		// fmt.Printf("Address for %s is %s\n", peer, nodeAddress)
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
			fmt.Println("Fehler beim Lesen des eingehenden Strings:", err)
		}
		fmt.Println("Received string:", receivedString)

		// Parsen des empfangenen Strings
		if strings.HasPrefix(receivedString, "Hello from node") {
			// Extrahieren der Zahl x aus der Nachricht
			x := receivedString[len("Hello from node") : len("Hello from node")+1]
			file, err = os.Open("ids.toml")
			if err != nil {
				log.Fatal("Error opening file:", err)
			}
			defer file.Close()
			if err := toml.NewDecoder(file).Decode(&configToml); err != nil {
				fmt.Println("Error reading config file:", err)
				return
			}
			// Aufrufen von QuerryAll mit der extrahierten Zahl x
			ctx := context.Background()
			_, err = connectToPeer(node, ctx, node.Addrs()[0].String(), configToml["node"+x])
			fmt.Println(err)
			QueryAll("node"+x, node, configToml["node"+x])
		} else {
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

			// Erstellen einer Nachricht
			message := Message{
				Time:    time,
				Content: content,
			}

			// Aktualisieren der Datei
			err = UpdateFile(message)
			if err != nil {
				fmt.Println("Fehler beim Aktualisieren der Datei:", err)
				return
			}
		}
	})

	node.SetStreamHandler("/application/1.0.0", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}
		// fmt.Println("Received string:", receivedString)
		if receivedString != Nodename && !contains(config.Peers, receivedString) {
			config.Peers = append(config.Peers, receivedString)
			// fmt.Println("Updated Peers:", config.Peers)
		}
	})

	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}

		// fmt.Printf("Address for %s is %s\n", peer, nodeAddress)
		ctx := context.Background()
		_, err = connectToPeer(node, ctx, node.Addrs()[0].String(), nodeAddress)
		if err != nil {
			fmt.Println("Error connecting to peer:", err)
		}
		for _, peer1 := range config.Peers {
			SendMessage(node, nodeAddress, peer1+"\n", "/application/1.0.0")
			SendMessage(node, nodeAddress, Nodename+"\n", "/application/1.0.0")
		}
	}
	time.Sleep(5 * time.Second)

	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}

		// fmt.Printf("Address for %s is %s\n", peer, nodeAddress)
		ctx := context.Background()
		_, err = connectToPeer(node, ctx, node.Addrs()[0].String(), nodeAddress)
		if err != nil {
			fmt.Println("Error connecting to peer:", err)
		}
		for _, peer1 := range config.Peers {
			SendMessage(node, nodeAddress, peer1+"\n", "/application/1.0.0")
			SendMessage(node, nodeAddress, Nodename+"\n", "/application/1.0.0")
			SendMessage(node, nodeAddress, "Hello from "+Nodename+"\n", "/chat/1.0.0")
		}
	}
	time.Sleep(5 * time.Second)

	message := Message{
		Time:    time.Now(),
		Content: "Message from " + Nodename,
	}
	UpdateFile(message)
	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}
		ctx := context.Background()
		_, err = connectToPeer(node, ctx, node.Addrs()[0].String(), nodeAddress)
		timeParts := strings.Split(message.Time.String(), " ")
		timeStr := strings.TrimSpace(timeParts[0] + "T" + timeParts[1] + "Z")
		SendMessage(node, nodeAddress, timeStr+"|Message from "+Nodename+"\n", "/chat/1.0.0")
	}

	saveConfig(&config, configPath)

	time.Sleep(60 * time.Second)
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
