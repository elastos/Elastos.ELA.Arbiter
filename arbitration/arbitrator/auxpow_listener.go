package arbitrator

import (
	"bytes"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA.SPV/bloom"
	spv "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA.SPV/interface/iutil"
	"github.com/elastos/Elastos.ELA.SideChain/auxpow"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/p2p/msg"
	ela "github.com/elastos/Elastos.ELA/core"
)

type AuxpowListener struct {
	ListenAddress string

	notifyQueue chan *notifyTask
}

func (l *AuxpowListener) Address() string {
	return l.ListenAddress
}

func (l *AuxpowListener) Type() ela.TransactionType {
	return ela.SideChainPow
}

func (l *AuxpowListener) Flags() uint64 {
	return spv.FlagNotifyInSyncing
}

func (l *AuxpowListener) Rollback(height uint32) {}

func (l *AuxpowListener) Notify(id common.Uint256, proof bloom.MerkleProof, tx ela.Transaction) {
	l.notifyQueue <- &notifyTask{id, &proof, &tx}
	log.Info("[Notify-Auxpow][", l.ListenAddress, "] find side aux pow transaction, hash:", tx.Hash().String())
	err := SpvService.SubmitTransactionReceipt(id, tx.Hash())
	if err != nil {
		return
	}
}

func (l *AuxpowListener) ProcessNotifyData(tasks []*notifyTask) {
	task := tasks[len(tasks)-1]
	log.Info("[Notify-ProcessNotifyData][", l.ListenAddress, "] process hash:", task.tx.Hash().String(), "len tasks:", len(tasks))
	err := SpvService.VerifyTransaction(*task.proof, *task.tx)
	if err != nil {
		log.Error("Verify transaction error: ", err)
		return
	}

	// Get Header from main chain
	header, err := SpvService.HeaderStore().Get(&task.proof.BlockHash)
	if err != nil {
		log.Error("can not get block from main chain")
		return
	}

	// Check if merkleroot is match
	merkleBlock := msg.MerkleBlock{
		Header:       header.BlockHeader,
		Transactions: task.proof.Transactions,
		Hashes:       task.proof.Hashes,
		Flags:        task.proof.Flags,
	}

	txId := task.tx.Hash()
	merkleBranch, err := bloom.GetTxMerkleBranch(merkleBlock, &txId)
	if err != nil {
		log.Error("can not get merkle branch")
		return
	}

	elaHeader, ok := header.BlockHeader.(*iutil.Header)
	if !ok {
		log.Error("invalid block header")
		return
	}
	// sideAuxpow serilze
	sideAuxpow := auxpow.SideAuxPow{
		SideAuxMerkleBranch: merkleBranch.Branches,
		SideAuxMerkleIndex:  merkleBranch.Index,
		SideAuxBlockTx:      *task.tx,
		MainBlockHeader:     *elaHeader.Header,
	}

	sideAuxpowBuf := bytes.NewBuffer([]byte{})
	err = sideAuxpow.Serialize(sideAuxpowBuf)
	if err != nil {
		log.Error("SideAuxpow serialize error: ", err)
		return
	}

	// send submit block
	payload, ok := task.tx.Payload.(*ela.PayloadSideChainPow)
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
		log.Info("Side node genesis block:", sideNode.GenesisBlock,
			"side aux pow tx genesis hash:", genesishashString)
		if sideNode.GenesisBlock == genesishashString {
			sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().
				GetSideChainManager().GetChain(sideNode.GenesisBlockAddress)
			if ok {
				currentHeight, err := sc.GetCurrentHeight()
				if err != nil {
					log.Error("Side chain GetCurrentHeight failed")
					return
				}
				if currentHeight == blockHeight {
					sideChain = sc
				} else {
					log.Warn("No need to submit auxpow, current side chain height:",
						currentHeight, " block height:", blockHeight)
					return
				}
			}
		}
	}

	if sideChain == nil {
		log.Error("Arbiter not find side chain")
		allChains := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetAllChains()
		for index, chain := range allChains {
			log.Error("Side chain", index, ":", chain.GetKey())
		}

		log.Error("Can not find side chain from genesis block hash: [", genesishashString, "]")
		return
	}

	sideChain.UpdateLastNotifySideMiningHeight(payload.SideGenesisHash)
	err = sideChain.SubmitAuxpow(genesishashString, blockhashString, sideAuxpowString)
	if err != nil {
		log.Error("[Notify-Auxpow] Submit SideAuxpow error: ", err)
		return
	}
	sideChain.UpdateLastSubmitAuxpowHeight(payload.SideGenesisHash)
}

func (l *AuxpowListener) start() {
	l.notifyQueue = make(chan *notifyTask, 10000)
	go func() {
		var tasks []*notifyTask
		for {
			select {
			case data, ok := <-l.notifyQueue:
				if ok {
					tasks = append(tasks, data)
					if len(tasks) >= 10000 {
						l.ProcessNotifyData(tasks)
						tasks = make([]*notifyTask, 0)
					}
				}
			default:
				if len(tasks) > 0 {
					//only deal with the last one task
					l.ProcessNotifyData(tasks)
					tasks = make([]*notifyTask, 0)
				}
				data, ok := <-l.notifyQueue
				if ok {
					tasks = append(tasks, data)
				}
			}
		}
	}()
}
