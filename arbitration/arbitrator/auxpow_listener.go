package arbitrator

import (
	"bytes"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.SideChain/auxpow"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
	"github.com/elastos/Elastos.ELA/bloom"
	ela "github.com/elastos/Elastos.ELA/core"
)

type AuxpowListener struct {
	ListenAddress string
}

func (l *AuxpowListener) Address() string {
	return l.ListenAddress
}

func (l *AuxpowListener) Type() ela.TransactionType {
	return ela.SideMining
}

func (l *AuxpowListener) Flags() uint64 {
	return 0
}

func (l *AuxpowListener) Rollback(height uint32) {

}

func (l *AuxpowListener) Notify(id common.Uint256, proof bloom.MerkleProof, tx ela.Transaction) {
	// Submit transaction receipt
	spvService.SubmitTransactionReceipt(id, tx.Hash())

	log.Info("[Notify] Receive sidemining transaction, hash:", tx.Hash().String())
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
	merkleBlock := msg.MerkleBlock{
		Header:       &header.Header,
		Transactions: proof.Transactions,
		Hashes:       proof.Hashes,
		Flags:        proof.Flags,
	}

	txId := tx.Hash()
	merkleBranch, err := bloom.GetTxMerkleBranch(merkleBlock, &txId)
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
	payload, ok := tx.Payload.(*ela.PayloadSideMining)
	if !ok {
		log.Error("Invalid payload type.")
		return
	}
	blockhashString := payload.SideBlockHash.String()
	genesishashString := payload.SideGenesisHash.String()
	blockHeight := payload.BlockHeight

	sideAuxpowData := sideAuxpowBuf.Bytes()
	sideAuxpowString := common.BytesToHexString(sideAuxpowData)

	var sideChain SideChain
	for _, sideNode := range config.Parameters.SideNodeList {
		if sideNode.GenesisBlock == genesishashString {
			sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetChain(sideNode.GenesisBlockAddress)
			if ok {
				currentHeight, err := sc.GetCurrentHeight()
				if err != nil {
					log.Error("Side chain GetCurrentHeight failed")
					return
				}
				if uint64(currentHeight) == blockHeight {
					sideChain = sc
				} else {
					log.Warn("No need to submit auxpow, current side chain height:", currentHeight, " block height:", blockHeight)
					return
				}
			}
		}
	}

	if sideChain == nil {
		log.Error("Can not faind side chain from genesis block hash: [", genesishashString, "]")
		return
	}

	err = sideauxpow.SubmitAuxpow(genesishashString, blockhashString, sideAuxpowString)
	if err != nil {
		log.Error("Submit SideAuxpow error: ", err)
		return
	}

	ArbitratorGroupSingleton.SyncFromMainNode()
	if ArbitratorGroupSingleton.GetCurrentArbitrator().IsOnDutyOfMain() {
		sideChain.StartSideChainMining()
	}
}
