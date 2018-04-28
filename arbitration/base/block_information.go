package base

import (
	"io"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA/core"
)

type PayloadInfo interface {
	Data(version byte) string
	Serialize(w io.Writer, version byte) error
	Deserialize(r io.Reader, version byte) error
}

type RegisterAssetInfo struct {
	Asset      *Asset
	Amount     string
	Controller string
}

type TransferAssetInfo struct {
}

type IssueTokenInfo struct {
	Proof string
}

type TransferCrossChainAssetInfo struct {
	AddressesMap map[string]uint64
}

/*type TxAttributeInfo struct {
	Usage AttributeUsage
	Data  string
}*/

/*type UTXOTxInputInfo struct {
	ReferTxID          string
	ReferTxOutputIndex uint16
	Sequence           uint32
	Address            string
	Value              string
}*/

/*type BalanceTxInputInfo struct {
	AssetID     string
	Value       Fixed64
	ProgramHash string
}*/

/*type TxoutputInfo struct {
	AssetID    string
	Value      string
	Address    string
	OutputLock uint32
}*/

type ProgramInfo struct {
	Code      string
	Parameter string
}

type TxoutputMap struct {
	Key   Uint256
	Txout []OutputInfo
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

/*type TransactionInfo struct {
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
	Confirmations     uint32 `json:",omitempty"`
	TxSize            uint32 `json:",omitempty"`
	Hash              string
}*/

type AttributeInfo struct {
	Usage AttributeUsage `json:"usage"`
	Data  string         `json:"data"`
}

type InputInfo struct {
	TxID     string `json:"txid"`
	VOut     uint16 `json:"vout"`
	Sequence uint32 `json:"sequence"`
}

type OutputInfo struct {
	Value      string `json:"value"`
	Index      uint32 `json:"n"`
	Address    string `json:"address"`
	AssetID    string `json:"assetid"`
	OutputLock uint32 `json:"outputlock"`
}

type TransactionInfo struct {
	TxId           string          `json:"txid"`
	Hash           string          `json:"hash"`
	Size           uint32          `json:"size"`
	VSize          uint32          `json:"vsize"`
	Version        uint32          `json:"version"`
	LockTime       uint32          `json:"locktime"`
	Inputs         []InputInfo     `json:"vin"`
	Outputs        []OutputInfo    `json:"vout"`
	BlockHash      string          `json:"blockhash"`
	Confirmations  uint32          `json:"confirmations"`
	Time           uint32          `json:"time"`
	BlockTime      uint32          `json:"blocktime"`
	TxType         TransactionType `json:"type"`
	PayloadVersion byte            `json:"payloadversion"`
	Payload        PayloadInfo     `json:"payload"`
	Attributes     []AttributeInfo `json:"attributes"`
	Programs       []ProgramInfo   `json:"programs"`
}

type BlockInfo struct {
	Hash            string
	BlockData       *BlockHead
	Transactions    []*TransactionInfo
	Confirminations uint32
	MinerInfo       string
}
