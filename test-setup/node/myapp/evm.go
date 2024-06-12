package main

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// DeployContract deploys a smart contract and returns the contract address
func DeployContract(bytecode []byte, dbFilePath string) (common.Address, error) {
	// Setup the EVM environment
	chainConfig := params.MainnetChainConfig
	gasLimit := uint64(3000000)           // Set a gas limit for contract deployment
	gasPrice := big.NewInt(1)             // Set a gas price
	address := common.HexToAddress("0x0") // This can be set to any valid address

	// Create a new state database
	filedb := NewFileDB(dbFilePath)
	statedb, _ := state.New(filedb)

	// Create a new EVM context
	context := vm.BlockContext{
		Coinbase:    common.Address{},
		BlockNumber: big.NewInt(1),
		Time:        big.NewInt(0),
		GasLimit:    gasLimit,
		Difficulty:  big.NewInt(0),
	}

	txContext := vm.TxContext{
		Origin:   address,
		GasPrice: gasPrice,
	}

	// Create a new EVM
	evm := vm.NewEVM(context, txContext, statedb, chainConfig, vm.Config{})

	// Deploy the contract
	sender := vm.AccountRef(address)
	contractAddress, _, gasUsed, err := evm.Create(sender, bytecode, statedb.GetBalance(address), gasLimit, gasPrice)

	if err != nil {
		log.Fatalf("Contract deployment failed: %v", err)
		return common.Address{}, err
	}

	log.Printf("Contract deployed at address: %s, gas used: %d", contractAddress.Hex(), gasUsed)
	return contractAddress, nil
}
