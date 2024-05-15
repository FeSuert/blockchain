package main

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func connectToPeer(h host.Host, peerID peer.ID, address string, debug ...bool) error {
	ctx := context.Background()
	peerAddr, err := ma.NewMultiaddr(address)
	if err != nil {
		return fmt.Errorf("error creating multiaddr: %v", err)
	}
	peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
	if err != nil {
		return fmt.Errorf("error creating peer info: %v", err)
	}

	if err := h.Connect(ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", peerInfo.ID, err)
	}
	if debug[0] == true {
		fmt.Printf("Successfully connected to peer %s\n", peerInfo.ID)
	}
	return nil
}
