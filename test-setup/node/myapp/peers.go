package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var GLOBAL_BLOCKCHAIN_PATH = "./data/blockchain.txt"

func ConnectToPeer(h host.Host, peerID peer.ID, address string, debug ...bool) error {

	ctx := context.Background()
	var peerInfo, err = GenerateMultiAddress(address)

	if err != nil {
		return fmt.Errorf("error generating multi address: %v", err)
	}

	if err := h.Connect(ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", peerInfo.ID, err)
	}

	if debug != nil {
		fmt.Printf("Successfully connected to peer %s\n", peerInfo.ID)
	}
	return nil
}

func GenerateMultiAddress(address string) (*peer.AddrInfo, error) {

	peerAddr, err := ma.NewMultiaddr(address)

	if err != nil {
		return nil, fmt.Errorf("error creating multiaddr: %v", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)

	if err != nil {
		return nil, fmt.Errorf("error creating peer info: %v", err)
	}

	return peerInfo, nil
}

func GetPeerIDFromPublicKey(pubKeyHex string) (peer.ID, error) {
	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("error decoding public key: %v", err)
	}
	libp2pPubKey, err := crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling public key: %v", err)
	}
	return peer.IDFromPublicKey(libp2pPubKey)
}

func RetrieveNodeFromPrivateKey(PrivKey string) (host.Host, error) {

	priv, err := GetPrivateKeyFromString(PrivKey)
	if err != nil {
		return nil, fmt.Errorf("error getting private key: %v", err)
	}

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

	node, err := GenerateNode(libp2pPrivKey)
	if err != nil {
		return nil, fmt.Errorf("error generating node: %v", err)
	}
	return node, nil
}

func GetPrivateKeyFromString(HexKey string) (ed25519.PrivateKey, error) {
	// Convert hex string to bytes
	privateKeyHex := strings.TrimLeft(HexKey, "0x")

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		panic(err)
	}
	// Create Ed25519 private key from seed
	priv := ed25519.NewKeyFromSeed(privateKeyBytes)
	return priv, nil
}

func GenerateNode(libp2pPrivKey crypto.PrivKey) (host.Host, error) {
	// Generate new node
	node, err := libp2p.New(
		libp2p.Identity(libp2pPrivKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/dns4/%s/tcp/8080", nodeName)),
	)
	if err != nil {
		panic(err)
	}
	return node, nil
}

func OpenPeerStream(node host.Host, config Config) {
	node.SetStreamHandler("/peers", func(s network.Stream) {
		config = handleIncomingMessage(s, config)

	})
}

func handleIncomingMessage(s network.Stream, config Config) Config {
	reader := bufio.NewReader(s)
	receivedString, err := reader.ReadString('\n')
	receivedString = strings.TrimSpace(receivedString)

	if err != nil {
		fmt.Println("Error reading incoming string:", err)
		return config
	}

	config = updatePeersList(receivedString, config)

	for _, peer := range config.Peers {
		sendMessageToPeer(node, peer, config)
	}
	return config
}

func updatePeersList(receivedString string, config Config) Config {
	if receivedString != nodeName && !Contains(config.Peers, receivedString) {
		config.Peers = append(config.Peers, receivedString)
	}
	return config
}

func sendMessageToPeer(node host.Host, peer string, config Config) {
	re := regexp.MustCompile(`\d+`)
	id, _ := strconv.Atoi(re.FindString(peer))
	peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
	address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
	ConnectToPeer(node, peerID, address)

	lines, _ := ReadAllLines(GLOBAL_BLOCKCHAIN_PATH)

	for _, line := range lines {
		SendMessage(node, address, line+"\n", "/blockchain")
	}
}
