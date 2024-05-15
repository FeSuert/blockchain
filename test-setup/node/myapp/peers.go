package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func ConnectToPeer(h host.Host, peerID peer.ID, address string, debug ...bool) error {

	ctx := context.Background()
	var peerInfo, err = GenerateMultiAddress(address)

	if err != nil {
		return fmt.Errorf("error generating multi address: %v", err)
	}

	if err := h.Connect(ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", peerInfo.ID, err)
	}

	if debug[0] == true {
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

	fmt.Println(PrivKey)
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
	fmt.Println(HexKey)
	privateKeyHex := strings.TrimLeft(HexKey, "0x")
	fmt.Println(privateKeyHex)
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	fmt.Println(privateKeyBytes)
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
