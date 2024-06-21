package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var fileMutex sync.Mutex

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func blockFromString(str string) (Block, error) {
	str = strings.TrimSpace(str)
	parts := strings.SplitN(str, "/", 4)
	if len(parts) != 4 {
		return Block{}, fmt.Errorf("invalid block string: %s", str)
	}

	currentID, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	prevID, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	leaderValue, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	transactions := strings.Split(strings.TrimSpace(parts[3]), ", ")

	receivedBlock := Block{
		ID:           currentID,
		PrevID:       prevID,
		LeaderValue:  leaderValue,
		Transactions: transactions,
	}
	return receivedBlock, nil
}

func writeAllLines(filePath string, lines []string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)

	defer file.Close()

	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	return writer.Flush()
}

func readAllLines(filePath string) ([]string, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(file)

	defer file.Close()

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func UpdateSortedMessages(message Message) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	filePath := "./data/sorted_messages.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}
	transaction := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s", message.Content.Sender, strconv.Itoa(message.Content.Nonce), message.Content.To, strconv.Itoa(message.Content.Amount), message.Content.Input, message.Content.Signature, message.Content.PublicKey)
	newLine := fmt.Sprintf("%s|%s", message.Time.Format(time.RFC3339), transaction)

	for _, line := range lines {
		if line == newLine {
			//fmt.Println("Duplicate message found, skipping:", newLine)
			return nil
		}
	}

	for _, peer := range config.Peers {
		//fmt.Println("Sending message to peer " + peer)
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

func UpdateTXResults(txHash, result string) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	filePath := "./data/tx_results.txt"
	lines, err := readAllLines(filePath)
	if err != nil {
		return err
	}
	newLine := fmt.Sprintf("%s|%s", txHash, result)

	for _, line := range lines {
		if line == newLine {
			fmt.Println("Duplicate message found, skipping:", newLine)
			return nil
		}
	}

	lines = append(lines, newLine)
	return writeAllLines(filePath, lines)
}

func int2bytes(i int) []byte {
	bi := big.NewInt(int64(i))
	return bi.Bytes()
}

func hexDecode(input string) []byte {
	if len(input) >= 2 && input[0:2] == "0x" {
		input = input[2:]
	}
	decoded, err := hex.DecodeString(input)
	if err != nil {
		fmt.Println("Input Error String:", input)
		log.Fatalf("failed to decode hex string: %v", err)
	}
	return decoded
}

func txBytes(tx TX) []byte {
	var data []byte
	senderBytes := hexDecode(tx.Sender)
	nonceBytes := int2bytes(tx.Nonce)
	toBytes := hexDecode(tx.To)
	amountBytes := int2bytes(tx.Amount)
	inputBytes := hexDecode(tx.Input)

	fmt.Printf("Go Sender Bytes: %x\n", senderBytes)
	fmt.Printf("Go Nonce Bytes: %x\n", nonceBytes)
	fmt.Printf("Go To Bytes: %x\n", toBytes)
	fmt.Printf("Go Amount Bytes: %x\n", amountBytes)
	//fmt.Printf("Go Input Bytes: %x\n", inputBytes)

	data = append(data, senderBytes...)
	data = append(data, nonceBytes...)
	if tx.To != "" {
		data = append(data, toBytes...)
		data = append(data, amountBytes...)
	}
	data = append(data, inputBytes...)
	return data
}

func hashTransaction(tx *TX) string {
	data := txBytes(*tx)

	hash := sha256.Sum256(data)
	txHash := hash[:]
	return "0x" + hex.EncodeToString(txHash)
}
