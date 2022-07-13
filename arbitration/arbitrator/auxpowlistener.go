package arbitrator

import (
	"bytes"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA.SPV/bloom"
	spv "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA.SPV/interface/iutil"
	"github.com/elastos/Elastos.ELA/common"
	elacommon "github.com/elastos/Elastos.ELA/core/types/common"
	it "github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/p2p/msg"
)

type AuxpowListener struct {
	ListenAddress string

	notifyQueue chan *notifyTask
}

func (l *AuxpowListener) Address() string {
	return l.ListenAddress
}

func (l *AuxpowListener) Type() elacommon.TxType {
	return elacommon.SideChainPow
}

func (l *AuxpowListener) Flags() uint64 {
	return spv.FlagNotifyInSyncing
}

func (l *AuxpowListener) Rollback(height uint32) {}

func (l *AuxpowListener) Notify(id common.Uint256, proof bloom.MerkleProof, tx it.Transaction) {
	l.notifyQueue <- &notifyTask{id, &proof, tx}
	log.Info("[Notify-Auxpow][", l.ListenAddress, "] find side aux pow transaction, hash:", tx.Hash().String())
	err := SpvService.SubmitTransactionReceipt(id, tx.Hash())
	if err != nil {
		return
	}
}

func (l *AuxpowListener) ProcessNotifyData(tasks []*notifyTask) {
	task := tasks[len(tasks)-1]
	log.Info("[Notify-ProcessNotifyData][", l.ListenAddress, "] process hash:", task.tx.Hash().String(), "len tasks:", len(tasks))
	err := SpvService.VerifyTransaction(*task.proof, task.tx)
	if err != nil {
		log.Error("verify transaction error: ", err)
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

	// serialize main chain tx
	buf := new(bytes.Buffer)
	if err := task.tx.Serialize(buf); err != nil {
		log.Error("invalid payload tx")
		return
	}

	// serialize merkle branch
	if err := common.WriteUint32(buf, uint32(len(merkleBranch.Branches))); err != nil {
		log.Error("serialize merkle branch count failed:", err.Error())
		return
	}
	for _, branch := range merkleBranch.Branches {
		err = branch.Serialize(buf)
		if err != nil {
			log.Error("serialize merkle branch failed:", err.Error())
			return
		}
	}
	if err := common.WriteUint32(buf, uint32(merkleBranch.Index)); err != nil {
		log.Error("serialize merkle branch index failed:", err.Error())
		return
	}

	// serialize ela header
	elaHeader := header.BlockHeader.(*iutil.Header)
	if err := elaHeader.Serialize(buf); err != nil {
		log.Error("invalid elaHeader")
		return
	}

	sideAuxpowString := common.BytesToHexString(buf.Bytes())

	p, ok := task.tx.Payload().(*payload.SideChainPow)
	if !ok {
		log.Error("invalid payload type")
		return
	}
	blockhashString := p.SideBlockHash.String()
	genesishashString := p.SideGenesisHash.String()
	blockHeight := p.BlockHeight

	var sideChain SideChain
	for _, sideNode := range config.Parameters.SideNodeList {
		log.Info("side node genesis block:", sideNode.GenesisBlock,
			"side aux pow tx genesis hash:", genesishashString)
		if sideNode.GenesisBlock == genesishashString {
			sc, ok := ArbitratorGroupSingleton.GetCurrentArbitrator().
				GetSideChainManager().GetChain(sideNode.GenesisBlockAddress)
			if ok {
				currentHeight, err := sc.GetCurrentHeight()
				if err != nil {
					log.Error("side chain GetCurrentHeight failed")
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
		log.Error("arbiter not find side chain")
		allChains := ArbitratorGroupSingleton.GetCurrentArbitrator().GetSideChainManager().GetAllChains()
		for index, chain := range allChains {
			log.Error("side chain", index, ":", chain.GetKey())
		}

		log.Error("can not find side chain from genesis block hash: [", genesishashString, "]")
		return
	}

	sideChain.UpdateLastNotifySideMiningHeight(p.SideGenesisHash)
	err = sideChain.SubmitAuxpow(genesishashString, blockhashString, sideAuxpowString)
	if err != nil {
		log.Error("[Notify-Auxpow] submit SideAuxpow error: ", err)
		return
	}
	sideChain.UpdateLastSubmitAuxpowHeight(p.SideGenesisHash)
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
