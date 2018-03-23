package sidechain

import (
	. "Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "Elastos.ELA.Arbiter/arbitration/base"
	"Elastos.ELA.Arbiter/common"
	"Elastos.ELA.Arbiter/common/config"
	tx "Elastos.ELA.Arbiter/core/transaction"
	"Elastos.ELA.Arbiter/crypto"
	"Elastos.ELA.Arbiter/rpc"
	spvMsg "SPVWallet/p2p/msg"
	spvWallet "SPVWallet/wallet"
	"bytes"
	"errors"
	"fmt"
)

type SideChainImpl struct {
	AccountListener
	key string

	currentConfig *config.SideNodeConfig
}

func (sc *SideChainImpl) GetKey() string {
	return sc.key
}

func (sc *SideChainImpl) getCurrentConfig() *config.SideNodeConfig {
	if sc.currentConfig == nil {
		for _, sideConfig := range config.Parameters.SideNodeList {
			if sc.GetKey() == sideConfig.GenesisBlockAddress {
				sc.currentConfig = sideConfig
				break
			}
		}
	}
	return sc.currentConfig
}

func (sc *SideChainImpl) GetCurrentHeight() (uint32, error) {
	return rpc.GetCurrentHeight(sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) GetBlockByHeight(height uint32) (*BlockInfo, error) {
	return rpc.GetBlockByHeight(height, sc.getCurrentConfig().Rpc)
}

func (sc *SideChainImpl) SendTransaction(info *TransactionInfo) error {
	infoDataReader := new(bytes.Buffer)
	err := info.Serialize(infoDataReader)
	if err != nil {
		return err
	}
	content := common.BytesToHexString(infoDataReader.Bytes())

	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", content), sc.currentConfig.Rpc)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func (sc *SideChainImpl) OnUTXOChanged(txinfo *TransactionInfo) error {
	//TODO　verify tx [jzh]

	txn, err := txinfo.ToTransaction()
	if err != nil {
		return err
	}
	withdrawInfo, err := sc.ParseUserWithdrawTransactionInfo(txn)
	if err != nil {
		return err
	}
	for _, info := range withdrawInfo {
		currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator()
		if err != nil {
			return err
		}
		withdrawTransaction, err := currentArbitrator.CreateWithdrawTransaction(sc.GetKey(), info.TargetProgramHash, info.Amount)
		if err != nil {
			return err
		}
		if withdrawTransaction == nil {
			return errors.New("Created an empty withdraw transaction.")
		}
		currentArbitrator.BroadcastWithdrawProposal(withdrawTransaction)
	}

	return nil
}

func (sc *SideChainImpl) CreateDepositTransaction(target common.Uint168, merkleBlock spvMsg.MerkleBlock, amount common.Fixed64) (*TransactionInfo, error) {
	var totalOutputAmount = amount // The total amount will be spend
	var txOutputs []TxoutputInfo   // The outputs in transaction

	toAddress, err := target.ToAddress()
	if err != nil {
		return nil, err
	}

	assetID := spvWallet.SystemAssetId
	txOutput := TxoutputInfo{
		AssetID:    assetID.String(),
		Value:      totalOutputAmount.String(),
		Address:    toAddress,
		OutputLock: uint32(0),
	}
	txOutputs = append(txOutputs, txOutput)

	// Create payload
	txPayloadInfo := new(IssueTokenInfo)

	// Create attributes
	spvInfo, err := merkleBlock.Serialize()
	if err != nil {
		return nil, err
	}
	txAttr := TxAttributeInfo{tx.SpvInfo, common.BytesToHexString(spvInfo)}
	attributes := make([]TxAttributeInfo, 0)
	attributes = append(attributes, txAttr)

	// Create program
	program := ProgramInfo{}
	return &TransactionInfo{
		TxType:        tx.IssueToken,
		Payload:       txPayloadInfo,
		Attributes:    attributes,
		UTXOInputs:    []UTXOTxInputInfo{},
		BalanceInputs: []BalanceTxInputInfo{},
		Outputs:       txOutputs,
		Programs:      []ProgramInfo{program},
		LockTime:      uint32(0),
	}, nil
}

func (sc *SideChainImpl) IsTransactionValid(transactionHash common.Uint256) (bool, error) {
	return false, nil
}

func (sc *SideChainImpl) ParseUserWithdrawTransactionInfo(txn *tx.Transaction) ([]*WithdrawInfo, error) {

	var result []*WithdrawInfo
	txAttribute := txn.Attributes
	for _, txAttr := range txAttribute {
		if txAttr.Usage == tx.TargetPublicKey {
			// Get public key
			keyBytes := txAttr.Data[0 : len(txAttr.Data)-1]
			key, err := crypto.DecodePoint(keyBytes)
			if err != nil {
				return nil, err
			}
			targetProgramHash, err := StandardAcccountPublicKeyToProgramHash(key)
			if err != nil {
				return nil, err
			}
			attrIndex := txAttr.Data[len(txAttr.Data)-1 : len(txAttr.Data)]
			for index, output := range txn.Outputs {
				if bytes.Equal([]byte{byte(index)}, attrIndex) {
					info := &WithdrawInfo{
						TargetProgramHash: *targetProgramHash,
						Amount:            output.Value,
					}
					result = append(result, info)
					break
				}
			}
		}
	}

	return result, nil
}

func init() {
	currentArbitrator := ArbitratorGroupSingleton.GetCurrentArbitrator().(*ArbitratorImpl)

	for _, sideConfig := range config.Parameters.SideNodeList {
		side := &SideChainImpl{
			key:           sideConfig.GenesisBlockAddress,
			currentConfig: sideConfig,
		}

		currentArbitrator.AddChain(sideConfig.GenesisBlockAddress, side)
	}
}
