package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Time    time.Time `json:"time"`
	Content TX        `json:"TX"`
}

type TX struct {
	Sender    string `json:"sender"`
	Nonce     int    `json:"nonce"`
	To        string `json:"to"` // Note: Adjust this field if the "to" key is present in your JSON.
	Amount    int    `json:"amount"`
	Input     string `json:"input"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
}

var sortedMessagesMutex sync.Mutex

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

func parseTransactionFromJSON(line string) (TX, error) {
	var tx TX

	// Replace single quotes with double quotes to make it a valid JSON string
	line = strings.ReplaceAll(line, "'", "\"")

	// Parse the JSON string into the TX struct
	err := json.Unmarshal([]byte(line), &tx)
	if err != nil {
		return tx, err
	}

	return tx, nil
}

func parseTransactionFromLine(line string) (TX, error) {
	parts := strings.Split(line, ",")
	if len(parts) != 7 {
		return TX{}, errors.New("invalid input line, expected 7 parts")
	}

	nonce, err := strconv.Atoi(parts[1])
	if err != nil {
		return TX{}, errors.New("invalid nonce value")
	}

	amount, err := strconv.Atoi(parts[3])
	if err != nil {
		return TX{}, errors.New("invalid amount value")
	}

	tx := TX{
		Sender:    parts[0],
		Nonce:     nonce,
		To:        parts[2],
		Amount:    amount,
		Input:     parts[4],
		Signature: parts[5],
		PublicKey: parts[6],
	}

	return tx, nil
}

func validateTransaction(transaction TX) (bool, error) {
	if !checkSender(transaction.Sender, transaction.PublicKey) {
		return false, errors.New("invalid sender")
	}
	if !checkNonce(transaction.Sender, transaction.Nonce) {
		return false, errors.New("used nonce")
	}
	// if !checkAmount(transaction.Sender, transaction.Amount) {
	// 	return false, errors.New("insufficient funds")
	// }
	return true, nil
}

func checkAmount(sender string, amount int) bool {
	if amount < 0 {
		return false
	}
	savedAmount := config.Addresses[sender]
	if savedAmount < amount {
		return false
	} else {
		return true
	}
}
func checkNonce(sender string, nonce int) bool {
	filePath := "./data/blockchain.txt"
	lines, _ := readAllLines(filePath)

	for _, line := range lines {
		block, err := blockFromString(line)
		if err != nil {
			fmt.Println("Error parsing block:", err)
			return false
		}
		for _, transaction := range block.Transactions {
			parts := strings.SplitN(transaction, "|", 2)
			tx, err := parseTransactionFromLine(parts[1])
			if err != nil {
				fmt.Println("Error parsing transaction TRANSACTION:", err)
				return false
			}
			if tx.Sender == sender {
				if tx.Nonce == nonce {
					return false
				}
			}
		}
	}
	return true
}
func checkSender(sender string, publicKey string) bool {
	if strings.HasPrefix(publicKey, "0x") {
		publicKey = publicKey[2:]
	}
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		fmt.Println("Error decoding public key:", err)
		return false
	}
	hash := sha256.Sum256(publicKeyBytes)
	derivedAddress := hash[:20]
	derivedAddressHex := hex.EncodeToString(derivedAddress)
	return derivedAddressHex == sender
}
