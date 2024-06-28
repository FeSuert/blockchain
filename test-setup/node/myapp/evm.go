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

	// hooks := &tracing.Hooks{
	// 	OnTxStart: func(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	// 		fmt.Printf("TxStart Hook: VM=%+v, Tx=%+v, From=%s\n", vm, tx, from.Hex())
	// 	},
	// 	OnTxEnd: func(receipt *types.Receipt, err error) {
	// 		fmt.Printf("TxEnd Hook: Receipt=%+v, Error=%v\n", receipt, err)
	// 	},
	// 	OnEnter: func(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// 		fmt.Printf("Enter Hook: Depth=%d, Type=%d, From=%s, To=%s, Input=%x, Gas=%d, Value=%s\n", depth, typ, from.Hex(), to.Hex(), input, gas, value.String())
	// 	},
	// 	OnExit: func(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	// 		fmt.Printf("Exit Hook: Depth=%d, Output=%x, GasUsed=%d, Error=%v, Reverted=%t\n", depth, output, gasUsed, err, reverted)
	// 	},
	// 	OnOpcode: func(pc uint64, op byte, gas uint64, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	// 		fmt.Printf("Opcode Hook: PC=%d, OpCode=%x, Gas=%d, Cost=%d, Depth=%d, Error=%v\n", pc, op, gas, cost, depth, err)
	// 	},
	// 	OnFault: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	// 		fmt.Printf("Fault Hook: PC=%d, OpCode=%x, Gas=%d, Cost=%d, Depth=%d, Error=%v\n", pc, op, gas, cost, depth, err)
	// 	},
	// 	OnGasChange: func(old, new uint64, reason tracing.GasChangeReason) {
	// 		fmt.Printf("Gas Change Hook: Old=%d, New=%d, Reason=%s\n", old, new, reason)
	// 	},
	// 	OnBlockchainInit: func(chainConfig *params.ChainConfig) {
	// 		fmt.Printf("Blockchain Init Hook: ChainConfig=%+v\n", chainConfig)
	// 	},
	// 	OnClose: func() {
	// 		fmt.Println("Close Hook")
	// 	},
	// 	OnBlockStart: func(event tracing.BlockEvent) {
	// 		fmt.Printf("Block Start Hook: Event=%+v\n", event)
	// 	},
	// 	OnBlockEnd: func(err error) {
	// 		fmt.Printf("Block End Hook: Error=%v\n", err)
	// 	},
	// 	OnSkippedBlock: func(event tracing.BlockEvent) {
	// 		fmt.Printf("Skipped Block Hook: Event=%+v\n", event)
	// 	},
	// 	OnGenesisBlock: func(genesis *types.Block, alloc types.GenesisAlloc) {
	// 		fmt.Printf("Genesis Block Hook: Block=%+v, Alloc=%+v\n", genesis, alloc)
	// 	},
	// 	OnSystemCallStart: func() {
	// 		fmt.Println("System Call Start Hook")
	// 	},
	// 	OnSystemCallEnd: func() {
	// 		fmt.Println("System Call End Hook")
	// 	},
	// 	OnBalanceChange: func(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	// 		fmt.Printf("Balance Change Hook: Address=%s, Previous=%s, New=%s, Reason=%s\n", addr.Hex(), prev.String(), new.String(), reason.String())
	// 	},
	// 	OnNonceChange: func(addr common.Address, prev, new uint64) {
	// 		fmt.Printf("Nonce Change Hook: Address=%s, Previous=%d, New=%d\n", addr.Hex(), prev, new)
	// 	},
	// 	OnCodeChange: func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	// 		fmt.Printf("Code Change Hook: Address=%s, PreviousCodeHash=%x, PreviousCode=%x, CodeHash=%x, Code=%x\n", addr.Hex(), prevCodeHash, prevCode, codeHash, code)
	// 	},
	// 	OnStorageChange: func(addr common.Address, slot common.Hash, prev, new common.Hash) {
	// 		fmt.Printf("Storage Change Hook: Address=%s, Slot=%x, Previous=%x, New=%x\n", addr.Hex(), slot, prev, new)
	// 	},
	// 	OnLog: func(log *types.Log) {
	// 		fmt.Printf("Log Hook: Log=%+v\n", log)
	// 	},
	// }

	evmConfig := vm.Config{
		//Tracer: hooks,
	}

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
		fmt.Printf("EVM: %+v\n", evm)
		fmt.Printf("Caller: %+v\n", caller)
		fmt.Printf("ToAddr: %+v\n", toAddr)
		fmt.Printf("Input: %x\n", input)
		fmt.Printf("Gas: %d\n", uint64(1000000))
		fmt.Printf("Value: %+v\n", value)

		// Execute the transaction in the EVM
		ret, _, err := evm.Call(caller, toAddr, input, uint64(1000000), uint256.NewInt(0))
		if err != nil {
			return "", fmt.Errorf("EVM call error: %v", err)
		}
		fmt.Println("Return Value:", ret)
		fmt.Println("Return Value as HEX:", common.Bytes2Hex(ret))
		return common.Bytes2Hex(ret), nil
	}
}
