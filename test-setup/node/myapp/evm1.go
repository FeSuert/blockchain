package main

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func deployContract(evm *vm.EVM, caller vm.ContractRef, code []byte, gas uint64, value *big.Int) (common.Address, error) {
	_, contractAddr, _, err := evm.Create(caller, code, gas, value)
	return contractAddr, err
}

func setupEVMEnvironment(
	coinbase common.Address,
	gasLimit uint64,
	blockNumber *big.Int,
	time uint64,
	difficulty *big.Int,
	baseFee *big.Int,
	stateDB *state.StateDB,
	chainConfig *params.ChainConfig,
	evmConfig vm.Config,
) *vm.EVM {

	blockCtx := vm.BlockContext{
		Coinbase:    coinbase,
		GasLimit:    gasLimit,
		BlockNumber: blockNumber,
		Time:        time,
		Difficulty:  difficulty,
		BaseFee:     baseFee,
	}

	txCtx := vm.TxContext{
		// Fill with necessary transaction context fields
	}

	evm := vm.NewEVM(blockCtx, txCtx, stateDB, chainConfig, evmConfig)
	return evm
}

func calculateStateDigest(stateDB *state.StateDB) string {
	hash := stateDB.IntermediateRoot(true).Bytes()
	return hex.EncodeToString(hash)
}
