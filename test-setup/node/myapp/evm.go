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
	// Set up the chain configuration with the necessary updates
	var (
		shanghaiTime = uint64(0)
		cancunTime   = uint64(0)
	)
	chainConfig := &params.ChainConfig{
		ChainID:                       big.NewInt(1),
		HomesteadBlock:                new(big.Int),
		DAOForkBlock:                  new(big.Int),
		DAOForkSupport:                false,
		EIP150Block:                   new(big.Int),
		EIP155Block:                   new(big.Int),
		EIP158Block:                   new(big.Int),
		ByzantiumBlock:                new(big.Int),
		ConstantinopleBlock:           new(big.Int),
		PetersburgBlock:               new(big.Int),
		IstanbulBlock:                 new(big.Int),
		MuirGlacierBlock:              new(big.Int),
		BerlinBlock:                   new(big.Int),
		LondonBlock:                   new(big.Int),
		ArrowGlacierBlock:             nil,
		GrayGlacierBlock:              nil,
		TerminalTotalDifficulty:       big.NewInt(0),
		TerminalTotalDifficultyPassed: true,
		MergeNetsplitBlock:            nil,
		ShanghaiTime:                  &shanghaiTime,
		CancunTime:                    &cancunTime,
	}

	evmConfig := vm.Config{}

	// Assign the myCanTransfer and myTransfer functions
	blockCtx.CanTransfer = myCanTransfer
	blockCtx.Transfer = myTransfer

	evm := vm.NewEVM(blockCtx, txCtx, stateDB, chainConfig, evmConfig)

	// Debugging: Print EVM initialization details
	// fmt.Printf("EVM Initialized with BlockContext: %+v and TxContext: %+v\n", blockCtx, txCtx)

	return evm
}

func myCanTransfer(stateDB vm.StateDB, from common.Address, amount *uint256.Int) bool {
	return stateDB.GetBalance(from).Cmp(amount) >= 0
}

func myTransfer(stateDB vm.StateDB, from, to common.Address, amount *uint256.Int) {
	stateDB.SubBalance(from, amount, 0)
	stateDB.AddBalance(to, amount, 0)
}

func executeTransaction(evm *vm.EVM, tx TX) (string, error) {
	// Convert sender address
	caller := vm.AccountRef(common.HexToAddress(tx.Sender))

	if tx.Input == "" {
		fmt.Println("Not a smart contract transaction")
		return "", nil
	}
	// Convert input data
	input := common.FromHex(tx.Input)

	// Convert amount to *uint256.Int
	value := uint256.MustFromBig(big.NewInt(int64(tx.Amount)))

	if evm == nil {
		return "", fmt.Errorf("evm instance is nil")
	}
	if evm.StateDB == nil {
		return "", fmt.Errorf("evm.StateDB is nil")
	}

	// Check if this is a contract creation transaction
	if tx.To == "" {
		// Debugging: Print all values before the call
		// fmt.Printf("EVM: %+v\n", evm)
		// fmt.Printf("Caller: %+v\n", caller)
		// //fmt.Printf("Input: %x\n", input)
		// fmt.Printf("Gas: %d\n", uint64(1000000))
		// fmt.Printf("Value: %+v\n", value)

		// Ensure the caller account exists in stateDB
		if !evm.StateDB.Exist(caller.Address()) {
			evm.StateDB.CreateAccount(caller.Address())
			evm.StateDB.AddBalance(caller.Address(), value, 0)
		}

		// Check if caller account was created successfully
		if !evm.StateDB.Exist(caller.Address()) {
			return "", fmt.Errorf("failed to create caller account in stateDB")
		}

		// Debugging: Check balance of caller
		callerBalance := evm.StateDB.GetBalance(caller.Address())
		fmt.Printf("Caller balance: %+v\n", callerBalance)

		// Create the contract in the EVM
		// fmt.Println("3. EVM Here")

		if caller.Address() == (common.Address{}) {
			return "", fmt.Errorf("caller address is nil or invalid")
		}
		if len(input) == 0 {
			return "", fmt.Errorf("input code is empty")
		}
		if value == nil {
			return "", fmt.Errorf("value is nil")
		}

		// // Debugging: Print detailed state before Create call
		// fmt.Printf("EVM State before Create:\nCaller Address: %s\nInput: %x\nGas: %d\nValue: %+v\n", caller.Address().Hex(), input, uint64(1000000), value)

		// // Detailed debugging for Create call
		// fmt.Printf("Calling evm.Create with parameters:\nCaller: %+v\nInput: %x\nGas: %d\nValue: %+v\n", caller, input, uint64(1000000), value)

		_, contractAddr, _, err := evm.Create(caller, input, uint64(3000000), value)
		if err != nil {
			fmt.Printf("Create call failed: %v\n", err)
			return "", fmt.Errorf("EVM create error: %v", err)
		}
		fmt.Println("Contract Address:", contractAddr)
		return contractAddr.Hex(), nil
	} else {
		// Convert recipient address
		toAddr := common.HexToAddress(tx.To)

		// Check for nil recipient address
		if toAddr == (common.Address{}) {
			return "", fmt.Errorf("invalid recipient address")
		}

		// Debugging: Print all values before the call
		// fmt.Printf("EVM: %+v\n", evm)
		// fmt.Printf("Caller: %+v\n", caller)
		// fmt.Printf("ToAddr: %+v\n", toAddr)
		// fmt.Printf("Input: %x\n", input)
		// fmt.Printf("Gas: %d\n", uint64(1000000))
		// fmt.Printf("Value: %+v\n", value)

		// Execute the transaction in the EVM
		ret, _, err := evm.Call(caller, toAddr, input, uint64(1000000), value)
		if err != nil {
			return "", fmt.Errorf("EVM call error: %v", err)
		}
		return common.Bytes2Hex(ret), nil
	}
}
