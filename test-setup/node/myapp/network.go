package main

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	prt "github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

func createLibp2pHost(priv crypto.PrivKey, nodeName string) (host.Host, error) {
	return libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/dns4/%s/tcp/8080", nodeName)),
	)
}

func connectToPeer(h host.Host, peerID peer.ID, address string) error {
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
	//fmt.Printf("Successfully connected to peer %s\n", peerInfo.ID)
	return nil
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
		//fmt.Println("Error opening stream to peer:", peerInfo.ID, err)
		return
	}

	_, err = stream.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending message to peer:", peerInfo.ID, err)
	}
	defer stream.Close()
}

func initializeStreamHandlers(node host.Host, state *ConsensusState) {
	node.SetStreamHandler("/transactions", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		//fmt.Println("Received message:" + receivedString)
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
		}
		parts := strings.SplitN(receivedString, "|", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid input format:", receivedString)
			return
		}
		content := strings.TrimSpace(parts[1])
		transaction, err := parseTransactionFromLine(content)
		if err != nil {
			fmt.Println("Error parsing transaction NETWORK:", err, content)
			return
		}
		time, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			fmt.Println("Error parsing time:", err)
			return
		}

		message := Message{
			Time:    time,
			Content: transaction,
		}
		fmt.Println("Sender in network received string:" + transaction.Sender)
		err = UpdateFile(message)
		if err != nil {
			fmt.Println("Error updating file:", err)
			return
		}
	})
	node.SetStreamHandler("/peers", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		//fmt.Println("Received message:" + receivedString)

		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}

		if receivedString != nodeName && !contains(config.Peers, receivedString) {
			config.Peers = append(config.Peers, receivedString)
		}

		for _, peer := range config.Peers {
			//fmt.Println("Sending message to peer " + peer)
			re := regexp.MustCompile(`\d+`)
			id, _ := strconv.Atoi(re.FindString(peer))
			peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
			connectToPeer(node, peerID, address)

			filePath := "./data/blockchain.txt"
			lines, _ := readAllLines(filePath)

			for _, line := range lines {
				SendMessage(node, address, line+"\n", "/blockchain")
			}
		}
	})
	node.SetStreamHandler("/consensus", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		receivedString = strings.TrimSpace(receivedString)
		fmt.Println("Received message:" + receivedString + " Current leader Value:" + strconv.Itoa(state.ReceivedMinLeaderValue))
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}

		parts := strings.SplitN(receivedString, "|", 2)
		if len(parts) != 2 {
			fmt.Println("Invalid input format:", receivedString)
			return
		}
		currentBlockID, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		if currentBlockID < state.CurrentBlockID {
			return
		}
		if currentBlockID > state.CurrentBlockID {
			state.CurrentBlockID = currentBlockID
			for _, peer := range config.Peers {
				re := regexp.MustCompile(`\d+`)
				id, _ := strconv.Atoi(re.FindString(peer))
				peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
				address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
				connectToPeer(node, peerID, address)
				SendMessage(node, address, nodeName+"\n", "/peers")
			}
			return
		}

		leaderValue, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return
		}

		state.CurrentBlockID = currentBlockID
		if state.ReceivedMinLeaderValue > leaderValue {
			state.ReceivedMinLeaderValue = leaderValue
		}
	})
	node.SetStreamHandler("/blockchain", func(s network.Stream) {
		reader := bufio.NewReader(s)
		receivedString, err := reader.ReadString('\n')
		//fmt.Println("Received message:" + receivedString)
		if err != nil {
			fmt.Println("Error reading incoming string:", err)
			return
		}
		receivedBlock, err := blockFromString(receivedString)
		if err != nil {
			fmt.Println("Error parsing block:", err)
			return
		}

		state.CurrentBlockID, err = SaveBlock(receivedBlock, state, config, node)
		if err != nil {
			return
		}
	})
}

func connectToPeers(node host.Host, peers, miners []string) {
	for _, peer := range peers {
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, err := getPeerIDFromPublicKey(miners[id-1])

		if err != nil {
			fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
			continue
		}

		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)

		if err := connectToPeer(node, peerID, address); err != nil {
			fmt.Printf("Error connecting to peer %s: %v\n", peerID, err)
		}

		for _, knownPeer := range peers {
			if knownPeer != peer {
				SendMessage(node, address, knownPeer+"\n", "/peers")
			}
		}
		SendMessage(node, address, nodeName+"\n", "/peers")
	}
}
