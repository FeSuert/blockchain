package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
)

var (
	fileMutex sync.Mutex
)

type Message struct {
	Time    time.Time `json:"time"`
	Content string    `json:"content"`
}

// SaveTransaction updates the file with a new message and sends it to all peers.
// It checks if the message already exists in the file before appending it.
// If the message is unique, it sorts the lines based on the timestamp and writes them back to the file.
func SaveTransaction(message Message, config Config, node host.Host) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Read the current contents of the file
	filePath := "./data/sorted_messages.txt"
	lines, err := ReadAllLines(filePath)
	if err != nil {
		return err
	}

	// Format the new message in the correct format
	newLine := fmt.Sprintf("%s|%s", message.Time.Format(time.RFC3339), message.Content)

	// Check if the message already exists
	for _, line := range lines {
		if line == newLine {
			//fmt.Println("Duplicate message found, skipping:", newLine)
			return nil
		}
	}

	// Send the new message to all peers
	for _, peer := range config.Peers {
		//fmt.Println("Sending message to peer " + peer)
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, _ := GetPeerIDFromPublicKey(config.Miners[id-1])
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		ConnectToPeer(node, peerID, address)
		SendMessage(node, address, newLine+"\n", "/transactions")
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

func removeTransaction(targetLine string) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	filePath := "./data/sorted_messages.txt"
	targetLine = strings.TrimSpace(targetLine)
	lines, err := ReadAllLines(filePath)
	if err != nil {
		return err
	}

	var newLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, targetLine) {
			newLines = append(newLines, line)
		}
	}

	err = writeAllLines(filePath, newLines)
	if err != nil {
		fmt.Println("Error writing updated transactions:", err)
		return err
	}

	return nil
}
