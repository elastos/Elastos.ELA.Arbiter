package arbitrator

import (
	"errors"
	"math"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/core/types"
)

type SideChain interface {
	AccountListener
	P2PClientListener
	SideChainNode

	GetKey() string
	GetExchangeRate() (float64, error)

	SetLastUsedUtxoHeight(height uint32)
	GetLastUsedUtxoHeight() uint32
	GetLastUsedOutPoints() []types.OutPoint
	AddLastUsedOutPoints(ops []types.OutPoint)
	RemoveLastUsedOutPoints(ops []types.OutPoint)
	ClearLastUsedOutPoints()

	GetExistDepositTransactions(txs []string) ([]string, error)

	GetWithdrawTransaction(txHash string) (*WithdrawTxInfo, error)
}

type SideChainManager interface {
	GetChain(key string) (SideChain, bool)
	GetAllChains() []SideChain

	StartSideChainMining()
	CheckAndRemoveWithdrawTransactionsFromDB() error
}

type DbMainChainFunc struct {
}

func (dbFunc *DbMainChainFunc) GetAvailableUtxos(withdrawBank string) ([]*store.AddressUTXO, error) {
	utxos, err := store.DbCache.UTXOStore.GetAddressUTXOsFromGenesisBlockAddress(withdrawBank)
	if err != nil {
		return nil, errors.New("Get spender's UTXOs failed.")
	}
	var availableUTXOs []*store.AddressUTXO
	var currentHeight = store.DbCache.UTXOStore.CurrentHeight(store.QueryHeightCode)
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

func (dbFunc *DbMainChainFunc) GetMainNodeCurrentHeight() (uint32, error) {
	chainHeight, err := rpc.GetCurrentHeight(config.Parameters.MainNode.Rpc)
	if err != nil {
		return 0, err
	}
	return chainHeight, nil
}
