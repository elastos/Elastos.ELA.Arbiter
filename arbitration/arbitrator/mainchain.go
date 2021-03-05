package arbitrator

import (
	"errors"
	"math"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
)

type MainChain interface {
	CreateWithdrawTransaction(sideChain SideChain, withdrawTxs []*base.WithdrawTx,
		mcFunc MainChainFunc) (*types.Transaction, error)
	CreateFailedDepositTransaction(sideChain SideChain, failedDepositTxs []*base.FailedDepositTx,
		mcFunc MainChainFunc, sideHeight uint32) (*types.Transaction, error)

	BroadcastWithdrawProposal(txn *types.Transaction) error
	BroadcastSidechainIllegalData(data *payload.SidechainIllegalData) error
	ReceiveProposalFeedback(content []byte) error

	SyncMainChainCachedTxs() error
	CheckAndRemoveDepositTransactionsFromDB() error
	SyncChainData() uint32
}

type MainChainClient interface {
	OnReceivedProposal(id peer.PID, content []byte) error
}

type MainChainFunc interface {
	GetWithdrawUTXOsByAmount(withdrawBank string,
		fixed64 common.Fixed64) ([]*store.AddressUTXO, error)
	GetMainNodeCurrentHeight() (uint32, error)
	GetAmountByInputs(inputs []*types.Input) (common.Fixed64, error)
	GetReferenceAddress(txid string, index int) (string, error)
}

type MainChainFuncImpl struct {
}

func (dbFunc *MainChainFuncImpl) GetWithdrawUTXOsByAmount(
	withdrawBank string, amount common.Fixed64) ([]*store.AddressUTXO, error) {
	utxos, err := dbFunc.GetWithdrawAddressUTXOsByAmount(withdrawBank, amount)
	if err != nil {
		return nil, errors.New("get spender's UTXOs failed, err:" + err.Error())
	}
	var availableUTXOs []*store.AddressUTXO
	var currentHeight = store.DbCache.MainChainStore.CurrentHeight(
		store.QueryHeightCode)
	for _, utxo := range utxos {
		if utxo.Input.Sequence > 0 {
			if utxo.Input.Sequence >= currentHeight {
				continue
			}
			utxo.Input.Sequence = math.MaxUint32 - 1
		}
		availableUTXOs = append(availableUTXOs, utxo)
	}
	availableUTXOs = store.SortUTXOs(availableUTXOs)

	return availableUTXOs, nil
}

func (dbFunc *MainChainFuncImpl) GetWithdrawAddressUTXOsByAmount(
	genesisBlockAddress string, amount common.Fixed64) ([]*store.AddressUTXO, error) {
	utxoInfos, err := rpc.GetWithdrawUTXOsByAmount(genesisBlockAddress, amount,
		config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	var inputs []*store.AddressUTXO
	for _, utxoInfo := range utxoInfos {

		bytes, err := common.HexStringToBytes(utxoInfo.Txid)
		if err != nil {
			return nil, err
		}
		reversedBytes := common.BytesReverse(bytes)
		txid, err := common.Uint256FromBytes(reversedBytes)
		if err != nil {
			return nil, err
		}

		var op types.OutPoint
		op.TxID = *txid
		op.Index = uint16(utxoInfo.VOut)

		amount, err := common.StringToFixed64(utxoInfo.Amount)
		if err != nil {
			return nil, err
		}

		inputs = append(inputs, &store.AddressUTXO{
			Input: &types.Input{
				Previous: op,
				Sequence: 0,
			},
			Amount:              amount,
			GenesisBlockAddress: genesisBlockAddress,
		})
	}
	return inputs, nil
}

func (dbFunc *MainChainFuncImpl) GetMainNodeCurrentHeight() (uint32, error) {
	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, err
	}
	return chainHeight, nil
}

func (dbFunc *MainChainFuncImpl) GetAmountByInputs(
	inputs []*types.Input) (common.Fixed64, error) {
	amount, err := rpc.GetAmountByInputs(inputs, config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func (dbFunc *MainChainFuncImpl) GetReferenceAddress(txid string, index int) (string, error) {
	addr, err := rpc.GetReferenceAddress(txid, index, config.Parameters.MainNode.Rpc)
	if err != nil {
		return "", err
	}
	return addr, nil
}
