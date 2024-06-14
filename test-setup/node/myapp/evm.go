package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethState "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func initializeEVM(blockCtx vm.BlockContext, txCtx vm.TxContext, stateDB *ethState.StateDB) *vm.EVM {
	chainConfig := params.MainnetChainConfig
	evmConfig := vm.Config{}

	// Assign the myCanTransfer and myTransfer functions
	blockCtx.CanTransfer = myCanTransfer
	blockCtx.Transfer = myTransfer

	evm := vm.NewEVM(blockCtx, txCtx, stateDB, chainConfig, evmConfig)

	// Debugging: Print EVM initialization details
	fmt.Printf("EVM Initialized with BlockContext: %+v and TxContext: %+v\n", blockCtx, txCtx)

	return evm
}

func myCanTransfer(stateDB vm.StateDB, from common.Address, amount *uint256.Int) bool {
	return stateDB.GetBalance(from).Cmp(amount) >= 0
}

func myTransfer(stateDB vm.StateDB, from, to common.Address, amount *uint256.Int) {
	stateDB.SubBalance(from, amount, 0)
	stateDB.AddBalance(to, amount, 0)
}

func executeTransaction(evm *vm.EVM, tx TX) ([]byte, error) {
	// Convert sender address
	caller := vm.AccountRef(common.HexToAddress(tx.Sender))

	// Convert input data
	input := common.FromHex(tx.Input)

	// Convert amount to *uint256.Int
	value := uint256.MustFromBig(big.NewInt(int64(tx.Amount)))

	if evm == nil {
		return nil, fmt.Errorf("evm instance is nil")
	}
	if evm.StateDB == nil {
		return nil, fmt.Errorf("evm.StateDB is nil")
	}

	// Check if this is a contract creation transaction
	if tx.To == "" {
		// Debugging: Print all values before the call
		fmt.Printf("EVM: %+v\n", evm)
		fmt.Printf("Caller: %+v\n", caller)
		fmt.Printf("Input: %x\n", input)
		fmt.Printf("Gas: %d\n", uint64(1000000))
		fmt.Printf("Value: %+v\n", value)

		// Ensure the caller account exists in stateDB
		if !evm.StateDB.Exist(caller.Address()) {
			evm.StateDB.CreateAccount(caller.Address())
			evm.StateDB.AddBalance(caller.Address(), value, 0)
		}

		// Check if caller account was created successfully
		if !evm.StateDB.Exist(caller.Address()) {
			return nil, fmt.Errorf("failed to create caller account in stateDB")
		}

		// Debugging: Check balance of caller
		callerBalance := evm.StateDB.GetBalance(caller.Address())
		fmt.Printf("Caller balance: %+v\n", callerBalance)

		// Create the contract in the EVM
		fmt.Println("3. EVM Here")

		if caller.Address() == (common.Address{}) {
			return nil, fmt.Errorf("caller address is nil or invalid")
		}
		if len(input) == 0 {
			return nil, fmt.Errorf("input code is empty")
		}
		if value == nil {
			return nil, fmt.Errorf("value is nil")
		}

		// Debugging: Print detailed state before Create call
		fmt.Printf("EVM State before Create:\nCaller Address: %s\nInput: %x\nGas: %d\nValue: %+v\n", caller.Address().Hex(), input, uint64(1000000), value)

		// Detailed debugging for Create call
		fmt.Printf("Calling evm.Create with parameters:\nCaller: %+v\nInput: %x\nGas: %d\nValue: %+v\n", caller, input, uint64(1000000), value)

		_, contractAddr, _, err := evm.Create(caller, input, uint64(3000000), value)
		if err != nil {
			fmt.Printf("Create call failed: %v\n", err)
			return nil, fmt.Errorf("EVM create error: %v", err)
		}
		fmt.Println("4. EVM Here")
		return contractAddr.Bytes(), nil
	} else {
		// Convert recipient address
		toAddr := common.HexToAddress(tx.To)

		// Check for nil recipient address
		if toAddr == (common.Address{}) {
			return nil, fmt.Errorf("invalid recipient address")
		}

		// Debugging: Print all values before the call
		fmt.Printf("EVM: %+v\n", evm)
		fmt.Printf("Caller: %+v\n", caller)
		fmt.Printf("ToAddr: %+v\n", toAddr)
		fmt.Printf("Input: %x\n", input)
		fmt.Printf("Gas: %d\n", uint64(1000000))
		fmt.Printf("Value: %+v\n", value)

		// Execute the transaction in the EVM
		ret, _, err := evm.Call(caller, toAddr, input, uint64(1000000), value)
		if err != nil {
			return nil, fmt.Errorf("EVM call error: %v", err)
		}
		return ret, nil
	}
}
