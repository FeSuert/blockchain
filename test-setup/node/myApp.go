package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	//"github.com/gorilla/rpc/json"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peer"
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

func connectToPeer(h host.Host, ctx context.Context, hostAddr string, peerAddr string) error {
	// Konvertiere die Multiaddress des Hosts und des Peers in ein Multiaddr-Objekt
	hostMultiAddr, err := ma.NewMultiaddr(hostAddr)
	if err != nil {
		return err
	}
	peerMultiAddr, err := ma.NewMultiaddr(peerAddr)
	if err != nil {
		return err
	}

	// Konvertiere die Multiaddress des Peers in einen PeerID
	peerID, err := peer.AddrInfoFromP2pAddr(peerMultiAddr)
	if err != nil {
		return err
	}

	// Füge die Multiaddress des Hosts hinzu
	peerID.Addrs = append(peerID.Addrs, hostMultiAddr)

	// Verbinde sich mit dem Peer
	if err := h.Connect(ctx, *peerID); err != nil {
		return err
	}

	fmt.Println("Verbunden mit Peer:", peerID.ID)
	return nil
}

func handleError(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func Broadcast(string) {
	fmt.Println("Query all messages")
}

func QueryAll() {
	fmt.Println("Return all messages that were received")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Bitte geben Sie den Pfad zur Konfigurationsdatei als Argument ein.")
		return
	}
	// Den Konfigurationspfad aus den Programmargumenten abrufen
	configPath := os.Args[1]

	// Read node name
	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	nodeName := "node" + numbers

	// Konfigurationsstruktur erstellen
	var config Config

	// Konfigurationsdatei öffnen und einlesen
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Println("Fehler beim Öffnen der Konfigurationsdatei:", err)
		return
	}
	defer file.Close()

	// Konfigurationsdatei analysieren und Daten in die Konfigurationsstruktur einfügen
	if err := toml.NewDecoder(file).Decode(&config); err != nil {
		fmt.Println("Fehler beim Lesen der Konfigurationsdatei:", err)
		return
	}

	// Konfigurationswerte ausgeben
	fmt.Println("Peers:", config.Peers)
	fmt.Println("RPC Port:", config.RPCPort)
	fmt.Println("Send Port:", config.SendPort)

	node, err := libp2p.New(libp2p.ListenAddrStrings(fmt.Sprintf("/dns4/%s/tcp/8443", nodeName)))
	if err != nil {
		panic(err)
	}
	fmt.Println(node.Addrs())
	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	fmt.Println("libp2p node address:", addrs[0])
	// Erstelle den Pfad zur ids.toml-Datei
	filePath := "ids.toml"
	num, err := strconv.Atoi(numbers)
	time.Sleep(time.Duration(num+3) * time.Second)
	tree, err := toml.LoadFile(filePath)
	if err != nil {
		// If there's an error loading the file (which includes the file being empty or non-existent), initialize a new tree.
		tree, err = toml.TreeFromMap(map[string]interface{}{})
	}

	tree.Set(nodeName, addrs[0].String())

	// Open or create the file
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error opening file for writing:", err)
		return
	}
	defer f.Close()
	// Write the updated tree to the file
	if _, err := tree.WriteTo(f); err != nil {
		fmt.Println("Error writing TOML data to file:", err)
		return
	}

	time.Sleep(5 * time.Second)

	file, err = os.Open("ids.toml")
	if err != nil {
		log.Fatal("Fehler beim Öffnen der Datei:", err)
	}
	defer file.Close()

	// Parse die ids.toml-Datei
	var configToml IDs
	if err := toml.NewDecoder(file).Decode(&configToml); err != nil {
		fmt.Println("Fehler beim Lesen der Konfigurationsdatei:", err)
		return
	}

	// Iteriere über jeden String-Eintrag in config.peer
	for _, peer := range config.Peers {
		nodeAddress, exists := configToml[peer]
		if !exists {
			fmt.Printf("Entry for node '%s' not found.\n", peer)
			continue
		}

		fmt.Printf("Address for %s is %s\n", peer, nodeAddress)
		ctx := context.Background()
		connectToPeer(node, ctx, node.Addrs()[0].String(), nodeAddress)
	}
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
