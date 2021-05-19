package base

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type ProgramInfo struct {
	Code      string
	Parameter string
}

type TxoutputMap struct {
	Key   common.Uint256
	Txout []OutputInfo
}

type AmountMap struct {
	Key   common.Uint256
	Value common.Fixed64
}

type AttributeInfo struct {
	Usage types.AttributeUsage `json:"usage"`
	Data  string               `json:"data"`
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
	TxType         types.TxType    `json:"type"`
	PayloadVersion byte            `json:"payloadversion"`
	Payload        PayloadInfo     `json:"payload"`
	Attributes     []AttributeInfo `json:"attributes"`
	Programs       []ProgramInfo   `json:"programs"`
}

type BlockInfo struct {
	Hash              string        `json:"hash"`
	Confirmations     uint32        `json:"confirmations"`
	StrippedSize      uint32        `json:"strippedsize"`
	Size              uint32        `json:"size"`
	Weight            uint32        `json:"weight"`
	Height            uint32        `json:"height"`
	Version           uint32        `json:"version"`
	VersionHex        string        `json:"versionhex"`
	MerkleRoot        string        `json:"merkleroot"`
	Tx                []interface{} `json:"tx"`
	Time              uint32        `json:"time"`
	MedianTime        uint32        `json:"mediantime"`
	Nonce             uint32        `json:"nonce"`
	Bits              uint32        `json:"bits"`
	Difficulty        string        `json:"difficulty"`
	ChainWork         string        `json:"chainwork"`
	PreviousBlockHash string        `json:"previousblockhash"`
	NextBlockHash     string        `json:"nextblockhash"`
	AuxPow            string        `json:"auxpow"`
}

type PayloadInfo interface {
}

type RegisterAssetInfo struct {
	Asset      *payload.Asset
	Amount     string
	Controller string
}

type CoinbaseInfo struct {
	CoinbaseData string
}

type RechargeToSideChainInfoV0 struct {
	Proof                string
	MainChainTransaction string
}

type RechargeToSideChainInfoV1 struct {
	MainChainTransactionHash string `json:"mainchaintxhash"`
}

type CrossChainAssetInfo struct {
	CrossChainAddress string `json:"crosschainaddress"`
	OutputIndex       uint64 `json:"outputindex"`
	CrossChainAmount  string `json:"crosschainamount"`
}

type TransferCrossChainAssetInfo struct {
	CrossChainAssets []CrossChainAssetInfo `json:"crosschainassets"`
}

type TransferAssetInfo struct {
}

type UTXOInfo struct {
	AssetId       string `json:"assetid"`
	Txid          string `json:"txid"`
	VOut          uint32 `json:"vout"`
	Address       string `json:"address"`
	Amount        string `json:"amount"`
	Confirmations uint32 `json:"confirmations"`
	OutputLock    uint32 `json:"OutputLock"`
}

type WithdrawOutputInfo struct {
	CrossChainAddress string `json:"crosschainaddress"`
	CrossChainAmount  string `json:"crosschainamount"`
	OutputAmount      string `json:"outputamount"`
	TargetData        string `json:"targetdata"`
}

type WithdrawTxInfo struct {
	TxID             string                `json:"txid"`
	CrossChainAssets []*WithdrawOutputInfo `json:"crosschainassets"`
}

type SidechainIllegalDataInfo struct {
	IllegalType     uint8  `json:"illegaltype"`
	Height          uint32 `json:"height"`
	IllegalSigner   string `json:"illegalsigner"`
	Evidence        string `json:"evidence"`
	CompareEvidence string `json:"compareevidence"`
}
