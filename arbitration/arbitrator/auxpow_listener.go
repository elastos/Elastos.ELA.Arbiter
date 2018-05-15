package arbitrator

import (
	"bytes"

	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.SideChain/auxpow"
	common "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/bloom"
	ela "github.com/elastos/Elastos.ELA/core"
)

var auxpowListener AuxpowListener

type AuxpowListener struct {
}

func (l *AuxpowListener) Type() ela.TransactionType {
	return ela.SideMining
}

func (l *AuxpowListener) Confirmed() bool {
	return false
}

func (l *AuxpowListener) Rollback(height uint32) {

}

func (l *AuxpowListener) Notify(proof bloom.MerkleProof, tx ela.Transaction) {
	log.Info("Receive sidemining transaction, hash:", tx.Hash().String())
	err := spvService.VerifyTransaction(proof, tx)
	if err != nil {
		log.Error("Verify transaction error: ", err)
		return
	}

	// Get Header from main chain
	header, err := spvService.HeaderStore().GetHeader(&proof.BlockHash)
	if err != nil {
		log.Error("can not get block from main chain")
		return
	}

	// Check if merkleroot is match
	merkleBlock := bloom.MerkleBlock{
		Header:       header.Header,
		Transactions: proof.Transactions,
		Hashes:       proof.Hashes,
		Flags:        proof.Flags,
	}

	txId := tx.Hash()
	merkleBranch, err := merkleBlock.GetTxMerkleBranch(&txId)
	if err != nil {
		log.Error("can not get merkle branch")
		return
	}

	// sideAuxpow serilze
	sideAuxpow := auxpow.SideAuxPow{
		SideAuxMerkleBranch: merkleBranch.Branches,
		SideAuxMerkleIndex:  merkleBranch.Index,
		SideAuxBlockTx:      tx,
		MainBlockHeader:     header.Header,
	}

	sideAuxpowBuf := bytes.NewBuffer([]byte{})
	err = sideAuxpow.Serialize(sideAuxpowBuf)
	if err != nil {
		log.Error("SideAuxpow serialize error: ", err)
		return
	}

	// send submit block
	payloadData := tx.Payload.Data(ela.SideMiningPayloadVersion)
	blockhashData := payloadData[0:32]
	blockhashString := common.BytesToHexString(blockhashData)

	sideAuxpowData := sideAuxpowBuf.Bytes()
	sideAuxpowString := common.BytesToHexString(sideAuxpowData)

	err = sideauxpow.SubmitAuxpow(blockhashString, sideAuxpowString)
	if err != nil {
		log.Error("Submit SideAuxpow error: ", err)
		return
	}

	// Submit transaction receipt
	spvService.SubmitTransactionReceipt(tx.Hash())
}
