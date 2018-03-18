package base

import (
	. "Elastos.ELA.Arbiter/common"
	. "Elastos.ELA.Arbiter/core/transaction"
)

type PayloadInfo interface{}

type TxAttributeInfo struct {
	Usage TransactionAttributeUsage
	Data  string
}

type UTXOTxInputInfo struct {
	ReferTxID          string
	ReferTxOutputIndex uint16
	Sequence           uint32
	Address            string
	Value              string
}

type BalanceTxInputInfo struct {
	AssetID     string
	Value       Fixed64
	ProgramHash string
}

type TxoutputInfo struct {
	AssetID    string
	Value      string
	Address    string
	OutputLock uint32
}

type ProgramInfo struct {
	Code      string
	Parameter string
}

type TxoutputMap struct {
	Key   Uint256
	Txout []TxoutputInfo
}

type AmountMap struct {
	Key   Uint256
	Value Fixed64
}

type BlockHead struct {
	Version          uint32
	PrevBlockHash    string
	TransactionsRoot string
	Timestamp        uint32
	Bits             uint32
	Height           uint32
	Nonce            uint32

	Hash string
}

type TransactionInfo struct {
	TxType         TransactionType
	PayloadVersion byte
	Payload        PayloadInfo
	Attributes     []TxAttributeInfo
	UTXOInputs     []UTXOTxInputInfo
	BalanceInputs  []BalanceTxInputInfo
	Outputs        []TxoutputInfo
	LockTime       uint32
	Programs       []ProgramInfo

	AssetOutputs      []TxoutputMap
	AssetInputAmount  []AmountMap
	AssetOutputAmount []AmountMap
	Timestamp         uint32 `json:",omitempty"`
	Confirminations   uint32 `json:",omitempty"`
	TxSize            uint32 `json:",omitempty"`
	Hash              string
}

type BlockInfo struct {
	Hash            string
	BlockData       *BlockHead
	Transactions    []*TransactionInfo
	Confirminations uint32
	MinerInfo       string
}

func (trans *TransactionInfo) Serialize() ([]byte, error) {
	return nil, nil
}

func (trans *TransactionInfo) Deserialize(data []byte) error {
	return nil
}

func (trans *TransactionInfo) ConvertFrom(tx *Transaction) error {
	return nil
}

func (trans *TransactionInfo) ConvertTo() (*Transaction, error) {
	return nil, nil
}