package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Time    time.Time `json:"time"`
	Content string    `json:"content"`
}

var fileMutex sync.Mutex
var sortedMessagesMutex sync.Mutex

func UpdateFile(message Message) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	filePath := "./data/sorted_messages.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}

	newLine := fmt.Sprintf("%s|%s", message.Time.Format(time.RFC3339), message.Content)

	for _, line := range lines {
		if line == newLine {
			//fmt.Println("Duplicate message found, skipping:", newLine)
			return nil
		}
	}

	for _, peer := range config.Peers {
		fmt.Println("Sending message to peer " + peer)
		re := regexp.MustCompile(`\d+`)
		id, _ := strconv.Atoi(re.FindString(peer))
		peerID, _ := getPeerIDFromPublicKey(config.Miners[id-1])
		address := fmt.Sprintf("/dns4/%s/tcp/8080/p2p/%s", peer, peerID)
		connectToPeer(node, peerID, address)
		SendMessage(node, address, newLine+"\n", "/transactions")
	}

	lines = append(lines, newLine)

	sort.SliceStable(lines, func(i, j int) bool {
		time1, _ := time.Parse(time.RFC3339, strings.SplitN(lines[i], "|", 2)[0])
		time2, _ := time.Parse(time.RFC3339, strings.SplitN(lines[j], "|", 2)[0])
		return time1.Before(time2)
	})

	return writeAllLines(filePath, lines)
}

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

func removeTransaction(targetLine string) error {
	sortedMessagesMutex.Lock()
	defer sortedMessagesMutex.Unlock()

	filePath := "./data/sorted_messages.txt"
	targetLine = strings.TrimSpace(targetLine)
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}

	var newLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != targetLine {
			newLines = append(newLines, line)
		} else {
			//fmt.Println("Removing line:", targetLine)
		}
	}
	err = writeAllLines(filePath, newLines)
	if err != nil {
		fmt.Println("Error writing updated transactions:", err)
		return err
	}

	return nil
}
