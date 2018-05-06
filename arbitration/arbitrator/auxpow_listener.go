package arbitrator

import (
	"bytes"
	"fmt"

	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	i "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA.SideChain/auxpow"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/bloom"
	ela "github.com/elastos/Elastos.ELA/core"
)

var spv i.SPVService

type AuxpowListener struct {
	txType ela.TransactionType
}

func (l *AuxpowListener) Type() ela.TransactionType {
	return l.txType
}

func (l *AuxpowListener) Confirmed() bool {
	return false
}

func (l *AuxpowListener) Rollback(height uint32) {

}

func (l *AuxpowListener) Notify(proof bloom.MerkleProof, tx ela.Transaction) {
	log.Debug("Receive unconfirmed transaction hash:", tx.Hash().String())
	err := spv.VerifyTransaction(proof, tx)
	if err != nil {
		log.Error("Verify transaction error: ", err)
		return
	}

	// Get Header from main chain
	header, err := spv.Blockchain().GetHeader(proof.BlockHash)
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

	// convert branch, dirty!
	var branch []Uint256
	for _, v := range merkleBranch.Branches {
		n, _ := Uint256FromBytes(v[:])
		branch = append(branch, *n)
	}

	// convert block header, dirty!
	txBuf := bytes.NewBuffer([]byte{})
	err = tx.Serialize(txBuf)
	if err != nil {
		log.Error("can not serialize tx")
		return
	}
	sideAuxBlockTx := ela.Transaction{}
	err = sideAuxBlockTx.Deserialize(txBuf)
	if err != nil {
		fmt.Println(err)
		return
	}

	// convert BlockHeader, dirty!
	headerBuf := bytes.NewBuffer([]byte{})
	err = header.Header.Serialize(headerBuf)
	if err != nil {
		log.Error("can not serialize blockheader")
		return
	}
	mainBlockHeader := ela.Header{}
	err = mainBlockHeader.Deserialize(headerBuf)
	if err != nil {
		fmt.Println(err)
		return
	}

	// sideAuxpow serilze
	sideAuxpow := auxpow.SideAuxPow{
		SideAuxMerkleBranch: branch,
		SideAuxMerkleIndex:  merkleBranch.Index,
		SideAuxBlockTx:      sideAuxBlockTx,
		MainBlockHeader:     mainBlockHeader,
	}

	fmt.Println("sideAuxpow", sideAuxpow)
	sideAuxpowBuf := bytes.NewBuffer([]byte{})
	err = sideAuxpow.Serialize(sideAuxpowBuf)
	if err != nil {
		fmt.Println(err)
		return
	}
	// fmt.Println("sideAuxpowBuf", sideAuxpowBuf)

	// send submit block
	payloadData := tx.Payload.Data(ela.SideMiningPayloadVersion)
	blockhashData := payloadData[0:32]
	blockhashString := BytesToHexString(blockhashData)

	sideAuxpowData := sideAuxpowBuf.Bytes()
	sideAuxpowString := BytesToHexString(sideAuxpowData)

	fmt.Println("blockhashString", blockhashString)
	fmt.Println("sideAuxpowString", sideAuxpowString)

	sideauxpow.SubmitAuxpow(blockhashString, sideAuxpowString)

	// Submit transaction receipt
	spv.SubmitTransactionReceipt(tx.Hash())
}
