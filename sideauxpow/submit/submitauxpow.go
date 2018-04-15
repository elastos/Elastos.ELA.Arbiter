package submit

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"

	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
	"github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	"github.com/elastos/Elastos.ELA.Arbiter/core/transaction/payload"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow/blockinfo"
	"github.com/elastos/Elastos.ELA.SPV/bloom"
	tx "github.com/elastos/Elastos.ELA.SPV/core/transaction"
	i "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA.SPV/log"
	spvconfig "github.com/elastos/Elastos.ELA.SPV/spvwallet/config"
)

var spv i.SPVService

func StartSPVListener() {
	log.Init()

	var id = make([]byte, 8)
	var clientId uint64
	var err error
	rand.Read(id)
	binary.Read(bytes.NewReader(id), binary.LittleEndian, clientId)
	spv = i.NewSPVService(clientId, spvconfig.Values().SeedList)

	// Register account
	err = spv.RegisterAccount("EN1M19RYHuFPS91hNRzR15TNtoAUDhi7hk")
	if err != nil {
		log.Error("Register account error: ", err)
		os.Exit(0)
	}

	// Set on transaction confirmed callback
	spv.RegisterTransactionListener(&UnconfirmedListener{txType: tx.SideMining})

	// Start spv service
	spv.Start()
}

type UnconfirmedListener struct {
	txType tx.TransactionType
}

func (l *UnconfirmedListener) Type() tx.TransactionType {
	return l.txType
}

func (l *UnconfirmedListener) Confirmed() bool {
	return false
}

func (l *UnconfirmedListener) Notify(proof i.Proof, tx tx.Transaction) {
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
		BlockHeader:  header.Header,
		Transactions: proof.Transactions,
		Hashes:       proof.Hashes,
		Flags:        proof.Flags,
	}

	txId := tx.Hash()
	merkleBranch, err := merkleBlock.GetTxMerkleBranch(txId)
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
	sideAuxBlockTx := transaction.Transaction{}
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
	mainBlockHeader := blockinfo.Blockdata{}
	err = mainBlockHeader.Deserialize(headerBuf)
	if err != nil {
		fmt.Println(err)
		return
	}

	// sideAuxpow serilze
	sideAuxpow := blockinfo.SideAuxPow{
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
	payloadData := tx.Payload.Data(payload.SideMiningPayloadVersion)
	blockhashData := payloadData[0:32]
	blockhashString := BytesToHexString(blockhashData)

	sideAuxpowData := sideAuxpowBuf.Bytes()
	sideAuxpowString := BytesToHexString(sideAuxpowData)

	fmt.Println("blockhashString", blockhashString)
	fmt.Println("sideAuxpowString", sideAuxpowString)

	submitAuxpow(blockhashString, sideAuxpowString)

	// Submit transaction receipt
	spv.SubmitTransactionReceipt(*tx.Hash())
}

func submitAuxpow(blockhash string, submitauxpow string) error {
	fmt.Println("submitauxblock")
	params := make(map[string]string, 2)
	params["blockhash"] = blockhash
	params["sideauxpow"] = submitauxpow
	resp, err := rpc.CallAndUnmarshal("submitauxblock", params, config.Parameters.SideNodeList[0].Rpc)
	if err != nil {
		return err
	}

	fmt.Println(resp)
	return nil
}
