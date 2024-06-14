package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	ethState "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/holiman/uint256"
)

var (
	stateDB *ethState.StateDB
	db      ethdb.Database
)

type AccountState struct {
	Balance *big.Int `json:"balance"`
	Nonce   uint64   `json:"nonce"`
}

func initState(config Config) {
	// Open the database (in-memory for simplicity)
	db := rawdb.NewMemoryDatabase()

	// Initialize the state
	var err error
	stateDB, err = ethState.New(common.Hash{}, ethState.NewDatabase(db), nil)
	if err != nil {
		log.Fatalf("Failed to create state: %v", err)
	}

	// Initialize the state with predefined balances
	for addr, balance := range config.Addresses {
		address := common.HexToAddress(addr)
		amount := uint256.NewInt(uint64(balance))
		fmt.Println("Address:" + addr + "Balance:" + strconv.Itoa(balance))
		stateDB.SetBalance(address, amount, 0)
	}

	// Commit the initial state
	root, err := stateDB.Commit(0, true)
	if err != nil {
		log.Fatalf("Failed to commit initial state: %v", err)
	}

	err = stateDB.Database().TrieDB().Commit(root, true)
	if err != nil {
		log.Fatalf("Failed to commit trie: %v", err)
	}
}

func saveAccountState(stateDB *ethState.StateDB, address common.Address) error {
	balance := stateDB.GetBalance(address)
	nonce := stateDB.GetNonce(address)
	accountState := AccountState{
		Balance: balance.ToBig(),
		Nonce:   nonce,
	}
	data, err := json.Marshal(accountState)
	if err != nil {
		return err
	}

	return db.Put(address.Bytes(), data)
}

func loadAccountState(stateDB *ethState.StateDB, address common.Address) error {
	data, err := db.Get(address.Bytes())
	if err != nil {
		return err
	}

	var accountState AccountState
	err = json.Unmarshal(data, &accountState)
	if err != nil {
		return err
	}

	balance := new(uint256.Int)
	balance.SetFromBig(accountState.Balance)
	stateDB.SetBalance(address, balance, 0)
	stateDB.SetNonce(address, accountState.Nonce)
	return nil
}

func saveContractState(stateDB *ethState.StateDB, address common.Address) error {
	storage := stateDB.GetState(address, common.Hash{}) // Get the storage trie
	code := stateDB.GetCode(address)

	storageData, err := json.Marshal(storage)
	if err != nil {
		return err
	}

	err = db.Put(append(address.Bytes(), []byte(":storage")...), storageData)
	if err != nil {
		return err
	}

	err = db.Put(append(address.Bytes(), []byte(":code")...), code)
	if err != nil {
		return err
	}

	return nil
}

func loadContractState(stateDB *ethState.StateDB, address common.Address) error {
	storageData, err := db.Get(append(address.Bytes(), []byte(":storage")...))
	if err != nil {
		return err
	}

	var storage map[common.Hash]common.Hash
	err = json.Unmarshal(storageData, &storage)
	if err != nil {
		return err
	}

	for key, value := range storage {
		stateDB.SetState(address, key, value)
	}

	codeData, err := db.Get(append(address.Bytes(), []byte(":code")...))
	if err != nil {
		return err
	}

	stateDB.SetCode(address, codeData)
	return nil
}
