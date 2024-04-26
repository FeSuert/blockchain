package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	//"github.com/gorilla/rpc/json"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Peers    []string `toml:"peers"`
	RPCPort  int      `toml:"rpc_port"`
	SendPort int      `toml:"send_port"`
}

func connectToPeers(h host.Host, peers []string, defaultPort int) error {
	for _, peerName := range peers {
		ctx := context.Background()
		fullAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/dns4/%s/tcp/%d", peerName, defaultPort))
		if err != nil {
			log.Printf("Error creating multiaddress for %s: %v", peerName, err)
			continue
		}

		peerInfo := &peer.AddrInfo{
			ID:    h.ID(),
			Addrs: []multiaddr.Multiaddr{fullAddr},
		}

		log.Printf("Attempting to connect to %s", peerName)
		h.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)
		if err := h.Connect(ctx, *peerInfo); err != nil {
			log.Printf("Failed to connect to %s: %v", peerName, err)
			continue
		}

		log.Printf("Connected to %s", peerName)
	}

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

	re := regexp.MustCompile(`\d+`)
	numbers := re.FindString(configPath)
	nodeName := "node" + numbers
	hash := sha256.Sum256([]byte(nodeName))
	peerID := fmt.Sprintf("%x", hash)
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

	node, err := libp2p.New(libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d/p2p/%s", config.RPCPort, peerID)))
	if err != nil {
		panic(err)
	}
	fmt.Println(node.Addrs())

	// Connect to peers
	if err := connectToPeers(node, config.Peers, 12345); err != nil {
		log.Fatalf("Error connecting to peers: %v", err)
	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh

	if err := node.Close(); err != nil {
		panic(err)
	}
}
