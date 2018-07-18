package cs

import (
	"os"
	"testing"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA/core"
	"github.com/stretchr/testify/assert"
)

type ClientTestFunc struct {
}

func (client *ClientTestFunc) GetSideChainAndExchangeRate(genesisAddress string) (arbitrator.SideChain, float64, error) {
	return nil, 1.0, nil
}

func TestClientInit(t *testing.T) {
	config.InitMockConfig()
	log.Init(log.Path, log.Stdout)

	dataStore, err := store.OpenDataStore()
	if err != nil {
		log.Fatal("Data store open failed error: [s%]", err.Error())
		os.Exit(1)
	}
	store.DbCache = *dataStore
}

func TestCheckWithdrawTransaction(t *testing.T) {
	//create data
	genesisAddress := "XQd1DCi6H62NQdWZQhJCRnrPn7sF9CTjaU"
	address1 := "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
	txId1 := common.Uint256{11}
	txId2 := common.Uint256{21}
	assetId := common.Uint256{12}
	amount1 := common.Fixed64(10000)
	amount2 := common.Fixed64(9000)
	amount3 := common.Fixed64(-9000)
	amount4 := common.Fixed64(20000)
	amount5 := common.Fixed64(0)
	amount6 := common.Fixed64(90000)

	programHash1, _ := common.Uint168FromAddress(address1)

	input1 := Input{Previous: OutPoint{TxID: txId1, Index: 0}, Sequence: 0}
	output1 := Output{AssetID: assetId, Value: amount1, OutputLock: 0, ProgramHash: common.Uint168{}}
	input2 := Input{Previous: OutPoint{TxID: txId2, Index: 0}, Sequence: 0}
	output2 := Output{AssetID: assetId, Value: amount2, OutputLock: 0, ProgramHash: *programHash1}
	output3 := Output{AssetID: assetId, Value: amount5, OutputLock: 0, ProgramHash: *programHash1}
	input4 := Input{Previous: OutPoint{TxID: txId2, Index: 0}, Sequence: 0}
	output4 := Output{AssetID: assetId, Value: amount6, OutputLock: 0, ProgramHash: *programHash1}

	addressUtxo1 := &store.AddressUTXO{Input: &input2, Amount: &amount1, GenesisBlockAddress: genesisAddress}
	store.DbCache.UTXOStore.AddAddressUTXO(addressUtxo1)

	//create transfer cross chain asset transaction
	tx1 := &Transaction{
		TxType:         8,
		PayloadVersion: 0,
		Payload: &PayloadTransferCrossChainAsset{
			CrossChainAddresses: []string{address1},
			OutputIndexes:       []uint64{0},
			CrossChainAmounts:   []common.Fixed64{amount2},
		},
		Attributes: nil,
		Inputs:     []*Input{&input1},
		Outputs:    []*Output{&output1},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}
	store.DbCache.SideChainStore.AddSideChainTx(&base.SideChainTransaction{tx1.Hash().String(), genesisAddress, tx1, 10})

	//create withdraw transaction
	tx2 := &Transaction{
		TxType:         7,
		PayloadVersion: 0,
		Payload: &PayloadWithdrawFromSideChain{
			BlockHeight:                10,
			GenesisBlockAddress:        genesisAddress,
			SideChainTransactionHashes: []common.Uint256{tx1.Hash()},
		},
		Attributes: nil,
		Inputs:     []*Input{&input2},
		Outputs:    []*Output{&output2},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}

	//check withdraw transaction
	err := checkWithdrawTransaction(tx2, &ClientTestFunc{})
	assert.NoError(t, err)

	//create transfer cross chain asset transaction
	tx1 = &Transaction{
		TxType:         8,
		PayloadVersion: 0,
		Payload: &PayloadTransferCrossChainAsset{
			CrossChainAddresses: []string{address1},
			OutputIndexes:       []uint64{0},
			CrossChainAmounts:   []common.Fixed64{amount2},
		},
		Attributes: nil,
		Inputs:     []*Input{&input1},
		Outputs:    []*Output{&output1},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}
	store.DbCache.SideChainStore.AddSideChainTx(&base.SideChainTransaction{tx1.Hash().String(), genesisAddress, tx1, 10})

	//create withdraw transaction with utxo is not from genesis address account
	tx2 = &Transaction{
		TxType:         7,
		PayloadVersion: 0,
		Payload: &PayloadWithdrawFromSideChain{
			BlockHeight:                10,
			GenesisBlockAddress:        genesisAddress,
			SideChainTransactionHashes: []common.Uint256{tx1.Hash()},
		},
		Attributes: nil,
		Inputs:     []*Input{&input2},
		Outputs:    []*Output{&output2},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}

	store.DbCache.UTXOStore.DeleteUTXO(addressUtxo1.Input)
	//check withdraw transaction
	err = checkWithdrawTransaction(tx2, &ClientTestFunc{})
	assert.EqualError(t, err, "Check withdraw transaction failed, utxo is not from genesis address account")

	store.DbCache.UTXOStore.AddAddressUTXO(addressUtxo1)
	//create transfer cross chain asset transaction with corss chain amount less than 0
	tx1 = &Transaction{
		TxType:         8,
		PayloadVersion: 0,
		Payload: &PayloadTransferCrossChainAsset{
			CrossChainAddresses: []string{address1},
			OutputIndexes:       []uint64{0},
			CrossChainAmounts:   []common.Fixed64{amount3},
		},
		Attributes: nil,
		Inputs:     []*Input{&input1},
		Outputs:    []*Output{&output1},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}
	store.DbCache.SideChainStore.AddSideChainTx(&base.SideChainTransaction{tx1.Hash().String(), genesisAddress, tx1, 10})

	//create withdraw transaction
	tx2 = &Transaction{
		TxType:         7,
		PayloadVersion: 0,
		Payload: &PayloadWithdrawFromSideChain{
			BlockHeight:                10,
			GenesisBlockAddress:        genesisAddress,
			SideChainTransactionHashes: []common.Uint256{tx1.Hash()},
		},
		Attributes: nil,
		Inputs:     []*Input{&input2},
		Outputs:    []*Output{&output2},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}

	//check withdraw transaction
	err = checkWithdrawTransaction(tx2, &ClientTestFunc{})
	assert.EqualError(t, err, "Check withdraw transaction failed, cross chain amount less than 0")

	//create transfer cross chain asset transaction with corss chain amount more than output amount
	tx1 = &Transaction{
		TxType:         8,
		PayloadVersion: 0,
		Payload: &PayloadTransferCrossChainAsset{
			CrossChainAddresses: []string{address1},
			OutputIndexes:       []uint64{0},
			CrossChainAmounts:   []common.Fixed64{amount4},
		},
		Attributes: nil,
		Inputs:     []*Input{&input1},
		Outputs:    []*Output{&output1},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}
	store.DbCache.SideChainStore.AddSideChainTx(&base.SideChainTransaction{tx1.Hash().String(), genesisAddress, tx1, 10})

	//create withdraw transaction
	tx2 = &Transaction{
		TxType:         7,
		PayloadVersion: 0,
		Payload: &PayloadWithdrawFromSideChain{
			BlockHeight:                10,
			GenesisBlockAddress:        genesisAddress,
			SideChainTransactionHashes: []common.Uint256{tx1.Hash()},
		},
		Attributes: nil,
		Inputs:     []*Input{&input2},
		Outputs:    []*Output{&output2},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}

	//check withdraw transaction
	err = checkWithdrawTransaction(tx2, &ClientTestFunc{})
	assert.EqualError(t, err, "Check withdraw transaction failed, cross chain amount more than output amount")

	//create transfer cross chain asset transaction
	tx1 = &Transaction{
		TxType:         8,
		PayloadVersion: 0,
		Payload: &PayloadTransferCrossChainAsset{
			CrossChainAddresses: []string{address1},
			OutputIndexes:       []uint64{0},
			CrossChainAmounts:   []common.Fixed64{amount2},
		},
		Attributes: nil,
		Inputs:     []*Input{&input1},
		Outputs:    []*Output{&output1},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}
	store.DbCache.SideChainStore.AddSideChainTx(&base.SideChainTransaction{tx1.Hash().String(), genesisAddress, tx1, 10})

	//create withdraw transaction with cross chain count not equal withdraw output count
	tx2 = &Transaction{
		TxType:         7,
		PayloadVersion: 0,
		Payload: &PayloadWithdrawFromSideChain{
			BlockHeight:                10,
			GenesisBlockAddress:        genesisAddress,
			SideChainTransactionHashes: []common.Uint256{tx1.Hash()},
		},
		Attributes: nil,
		Inputs:     []*Input{&input2},
		Outputs:    []*Output{&output2, &output3},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}

	//check withdraw transaction
	err = checkWithdrawTransaction(tx2, &ClientTestFunc{})
	assert.EqualError(t, err, "Check withdraw transaction failed, cross chain count not equal withdraw output count")

	//create transfer cross chain asset transaction
	tx1 = &Transaction{
		TxType:         8,
		PayloadVersion: 0,
		Payload: &PayloadTransferCrossChainAsset{
			CrossChainAddresses: []string{address1},
			OutputIndexes:       []uint64{0},
			CrossChainAmounts:   []common.Fixed64{amount2},
		},
		Attributes: nil,
		Inputs:     []*Input{&input1},
		Outputs:    []*Output{&output1},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}
	store.DbCache.SideChainStore.AddSideChainTx(&base.SideChainTransaction{tx1.Hash().String(), genesisAddress, tx1, 10})

	//create withdraw transaction with input amount not equal output amount
	tx2 = &Transaction{
		TxType:         7,
		PayloadVersion: 0,
		Payload: &PayloadWithdrawFromSideChain{
			BlockHeight:                10,
			GenesisBlockAddress:        genesisAddress,
			SideChainTransactionHashes: []common.Uint256{tx1.Hash()},
		},
		Attributes: nil,
		Inputs:     []*Input{&input4},
		Outputs:    []*Output{&output4},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}

	//check withdraw transaction
	err = checkWithdrawTransaction(tx2, &ClientTestFunc{})
	assert.EqualError(t, err, "Check withdraw transaction failed, input amount not equal output amount")

	store.DbCache.UTXOStore.ResetDataStore()
	store.DbCache.SideChainStore.ResetDataStore()
	store.DbCache.MainChainStore.ResetDataStore()
}
