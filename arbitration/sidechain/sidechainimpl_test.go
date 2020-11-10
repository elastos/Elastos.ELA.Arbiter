package sidechain

import (
	"bytes"
	abtor "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/elanet/bloom"
)

func TestClientInit(t *testing.T) {
	config.InitMockConfig()
	log.Init("./log", 1, 32, 64)
}

func TestCheckWithdrawTransaction(t *testing.T) {
	testLoopTimes := 10000

	//create data
	proofStr := "5f894325400c9a12f4490da7bca9f4e32466f497a65aacb2dbfa29ac14619944b300000001000000010000005f894325400c9a12f4490da7bca9f4e32466f497a65aacb2dbfa29ac14619944fd83010800012245544d4751433561473131627752677553704357324e6b7950387a75544833486e3200010013353537373030363739313934373737393431300403229feeff99fa03357d09648a93363d1d01f234e61d04d10f93c9ad1aef3c150100feffffff737a4387ebf5315b74c508e40ba4f0179fc1d68bf76ce079b6bbf26e0fd2aa470100feffffff592c415c08ac1e1312d98cf6a28f68b62dd28ae964ed33af882b2d16b3a44a900100feffffff34255723e2249e8d965892edb9cd4cbbe27fa30e1292372a07206079dfad4a260100feffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b00000000000000002132a3f3d36f0db243743debee55155d5343322c2ab037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3782e43120000000000000000216fd749255076c304942d16a8023a63b504b6022f570200000100232103c3ffe56a4c68b4dfe91573081898cb9a01830e48b8f181de684e415ecfc0e098ac"

	proof := new(bloom.MerkleProof)
	byteProof, _ := common.HexStringToBytes(proofStr)
	proofReader := bytes.NewReader(byteProof)
	proof.Deserialize(proofReader)

	genesisAddress := "XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ"
	address1 := "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"

	genesisProgramHash, _ := common.Uint168FromAddress(genesisAddress)
	programHash1, _ := common.Uint168FromAddress(address1)
	txId1 := common.Uint256{11}
	assetId := common.Uint256{12}
	amount1 := common.Fixed64(10000)
	amount2 := common.Fixed64(9000)
	amount4 := common.Fixed64(1000)

	input1 := types.Input{Previous: types.OutPoint{TxID: txId1, Index: 0}, Sequence: 0}
	output1 := types.Output{AssetID: assetId, Value: amount1, OutputLock: 0, ProgramHash: *genesisProgramHash}
	output2 := types.Output{AssetID: assetId, Value: amount2, OutputLock: 0, ProgramHash: *genesisProgramHash}
	output3 := types.Output{AssetID: assetId, Value: amount4, OutputLock: 0, ProgramHash: *programHash1}

	//create deposit transaction
	tx := &types.Transaction{
		TxType:         6,
		PayloadVersion: 0,
		Payload: &payload.TransferAsset{},
		Attributes: nil,
		Inputs:     []*types.Input{&input1},
		Outputs:    []*types.Output{&output1, &output2, &output3},
		LockTime:   0,
		Programs:   nil,
		Fee:        0,
		FeePerKB:   0,
	}

	mcDataStore, err := store.OpenMainChainDataStore()
	if err != nil {
		t.Error("Open database error.")
	}
	fhDataStore, err := store.OpenFinishedTxsDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	arbitrator := abtor.ArbitratorImpl{}
	sideChainManager := &SideChainManagerImpl{make(map[string]abtor.SideChain)}
	side := &SideChainImpl{
	}
	sideChainManager.AddChain(genesisAddress, side)
	arbitrator.SetSideChainManager(sideChainManager)

	var txs []*base.MainChainTransaction
	for i := 0; i < testLoopTimes; i++ {
		txBytes := tx.Hash().Bytes()
		txBytes[28] = byte(i)
		txBytes[29] = byte(i >> 8)
		txBytes[30] = byte(i >> 16)
		txBytes[31] = byte(i >> 24)
		txHash, _ := common.Uint256FromBytes(txBytes)
		txs = append(txs, &base.MainChainTransaction{
			TransactionHash:     txHash.String(),
			GenesisBlockAddress: genesisAddress,
			Transaction:         tx,
		})
	}

	if err != nil {
		t.Error("AddMainChainTx error:", err)
		return
	}

	var finalTxHashes []string
	var genesisAddresses []string


	err = fhDataStore.AddSucceedDepositTxs(finalTxHashes, genesisAddresses)
	if err != nil {
		t.Error("Add succeed deposit tx failed")
	}
	err = mcDataStore.RemoveMainChainTxs(finalTxHashes, genesisAddresses)
	if err != nil {
		t.Error("Remove main chain tx failed")
	}

	mcDataStore.ResetDataStore()
	fhDataStore.ResetDataStore()
}

