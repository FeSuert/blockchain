package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
)

type ConsensusState struct {
	currentBlockID         int
	receivedMinLeaderValue int
	ownLeaderValue         int
}

func handleConsensusMessage(s network.Stream) {
	reader := bufio.NewReader(s)
	receivedString, err := reader.ReadString('\n')
	receivedString = strings.TrimSpace(receivedString)
	fmt.Println("Received Consensus Message:" + receivedString)
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
	if currentBlockID < state.currentBlockID {
		fmt.Println("Old ID received:", currentBlockID)
		return
	}
	if currentBlockID > state.currentBlockID {
		fmt.Println("Newer ID received:", currentBlockID)
		for _, peer := range config.Peers {
			re := regexp.MustCompile(`\d+`)
			id, _ := strconv.Atoi(re.FindString(peer))
			peerID, err := getPeerIDFromPublicKey(config.Miners[id-1])

			if err != nil {
				fmt.Printf("Error getting peer ID for peer %s: %v\n", peer, err)
				continue
			}
			address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
			SendMessage(node, address, nodeName+"\n", "/peers")
		}
		return
	}

	leaderValue, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		fmt.Println("Error converting string:", err)
		return
	}

	// Update state
	state.currentBlockID = currentBlockID
	if state.receivedMinLeaderValue > leaderValue {
		state.receivedMinLeaderValue = leaderValue
	}
}
