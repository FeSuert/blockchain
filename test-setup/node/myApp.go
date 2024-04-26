package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	//"github.com/gorilla/rpc/json"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Peers    []string `toml:"peers"`
	RPCPort  int      `toml:"rpc_port"`
	SendPort int      `toml:"send_port"`
}

func connectToPeers(h host.Host, peers []string, defaultPort int) error {
	for _, peerName := range peers {
		// Construct the multiaddress assuming all peers are using the same port
		ctx := context.Background()
		fullAddr, err := madns.DefaultResolver.LookupIPAddr(ctx, peerName)
		if err != nil {
			log.Printf("Error looking up IP address for %s: %v", peerName, err)
			return nil // or handle the error appropriately
		}

		ipAddr := fullAddr[0].IP.String()
		addrStr := fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, defaultPort)
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			log.Printf("Error creating multiaddress for %s: %v", peerName, err)
			return nil // or handle the error appropriately
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Printf("Error getting peer info from %s: %v", addr, err)
			continue // or return err to stop the process
		}

		fmt.Printf("Attempting to connect to %s\n", addr)
		h.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)
		if err := h.Connect(context.Background(), *peerInfo); err != nil {
			log.Printf("Failed to connect to %s: %v", addr, err)
			continue // or return err to stop the process
		}

		fmt.Printf("Connected to %s\n", addr)
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

	node, err := libp2p.New(libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", config.RPCPort)))
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
