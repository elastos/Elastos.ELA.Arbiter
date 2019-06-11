package sideauxpow

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

var (
	lock                          sync.RWMutex
	client                        *account.Client
	lastSendSideMiningHeightMap   map[common.Uint256]uint32
	lastNotifySideMiningHeightMap map[common.Uint256]uint32
	lastSubmitAuxpowHeightMap     map[common.Uint256]uint32
)

func GetLastSendSideMiningHeight(genesisBlockHash *common.Uint256) (uint32, bool) {
	lock.RLock()
	defer lock.RUnlock()
	height, ok := lastSendSideMiningHeightMap[*genesisBlockHash]
	return height, ok
}

func GetLastNotifySideMiningHeight(genesisBlockHash *common.Uint256) (uint32, bool) {
	lock.RLock()
	defer lock.RUnlock()
	height, ok := lastNotifySideMiningHeightMap[*genesisBlockHash]
	return height, ok
}

func GetLastSubmitAuxpowHeight(genesisBlockHash *common.Uint256) (uint32, bool) {
	lock.RLock()
	defer lock.RUnlock()
	height, ok := lastSubmitAuxpowHeightMap[*genesisBlockHash]
	return height, ok

}

func UpdateLastNotifySideMiningHeight(genesisBlockHash common.Uint256) {
	lock.Lock()
	defer lock.Unlock()
	lastNotifySideMiningHeightMap[genesisBlockHash] = *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()
}

func UpdateLastSubmitAuxpowHeight(genesisBlockHash common.Uint256) {
	lock.Lock()
	defer lock.Unlock()
	lastSubmitAuxpowHeightMap[genesisBlockHash] = *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()
}

func unmarshal(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		return err
	}
	return nil
}

func sideChainPowTransfer(sideNode *config.SideNodeConfig) error {
	log.Info("[sideChainPowTransfer] start")

	if sideNode.PayToAddr == "" {
		return errors.New("[sideChainPowTransfer] has no side aux pow paytoaddr")
	}
	resp, err := rpc.CallAndUnmarshal("createauxblock", rpc.Param("paytoaddress", sideNode.PayToAddr), sideNode.Rpc)
	if err != nil {
		log.Errorf("[sideChainPowTransfer] create aux block failed: %s", err)
		return err
	}
	if resp == nil {
		log.Info("[sideChainPowTransfer] create auxblock, nil ")
		return nil
	}

	type SideAuxBlock struct {
		GenesisHash       string `json:"genesishash"`
		Height            uint32 `json:"height"`
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		PreviousBlockHash string `json:"previousblockhash"`
	}

	sideAuxBlock := &SideAuxBlock{}

	err = unmarshal(resp, sideAuxBlock)
	if err != nil {
		return err
	}

	txType := types.SideChainPow

	sideGenesisHashData, _ := common.HexStringToBytes(sideAuxBlock.GenesisHash)
	sideBlockHashData, _ := common.HexStringToBytes(sideAuxBlock.Hash)

	sideGenesisHash, _ := common.Uint256FromBytes(sideGenesisHashData)
	sideBlockHash, _ := common.Uint256FromBytes(sideBlockHashData)

	log.Info("sideGenesisHash:", sideGenesisHash, "sideBlockHash:", sideBlockHash)
	// Create payload
	txPayload := &payload.SideChainPow{
		BlockHeight:     sideAuxBlock.Height,
		SideBlockHash:   *sideBlockHash,
		SideGenesisHash: *sideGenesisHash,
	}

	buf := new(bytes.Buffer)
	txPayload.Serialize(buf, payload.SideChainPowVersion)
	txPayload.Signature, err = arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().Sign(buf.Bytes()[0:68])
	if err != nil {
		return err
	}

	// create transaction
	if config.Parameters.SideAuxPowFee <= 0 {
		return errors.New("[sideChainPowTransfer] invalid side aux pow fee")
	}
	fee := common.Fixed64(config.Parameters.SideAuxPowFee)

	if sideNode.MiningAddr == "" {
		return errors.New("[sideChainPowTransfer] get side chain mining address failed:" + sideNode.MiningAddr)
	}

	programHash, err := common.Uint168FromAddress(sideNode.MiningAddr)
	if err != nil {
		return errors.New("[sideChainPowTransfer] invalid miningAddr")
	}
	codeHash := programHash.ToCodeHash()
	miningAccount := client.GetAccountByCodeHash(codeHash)
	if miningAccount == nil {
		return errors.New("[sideChainPowTransfer] not found miningAddr in keystore")
	}

	from := sideNode.MiningAddr
	script := miningAccount.RedeemScript

	txn, err := createAuxpowTransaction(txType, txPayload, from, &fee, script, *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight())
	if err != nil {
		return errors.New("[sideChainPowTransfer] create transaction failed: " + err.Error())
	}

	txnSigned, err := client.Sign(txn)
	if err != nil {
		return err
	}
	program := txnSigned.Programs[0]
	haveSign, needSign, _ := crypto.GetSignStatus(program.Code, program.Parameter)
	log.Debug("[sideChainPowTransfer] transaction successfully signed: ", haveSign, needSign)

	sideChainPowBuf := new(bytes.Buffer)
	txn.Serialize(sideChainPowBuf)
	content := common.BytesToHexString(sideChainPowBuf.Bytes())
	// log.Debug("Raw Sidemining transaction: ", content)

	// send transaction
	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return errors.New("[SendSideChainMining] sendrawtransaction failed: " + err.Error())
	}
	log.Info("[SendSideChainMining] End send Sidemining transaction:  genesis address [", sideNode.GenesisBlockAddress, "], result: ", result)

	lock.Lock()
	defer lock.Unlock()
	lastSendSideMiningHeightMap[*sideGenesisHash] = *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()

	log.Info("[sideChainPowTransfer] end")
	return nil
}

func StartSideChainMining(sideNode *config.SideNodeConfig) {
	err := sideChainPowTransfer(sideNode)
	if err != nil {
		log.Warn(err)
	}
}

func Init(c *account.Client) {
	client = c
	lastSendSideMiningHeightMap = make(map[common.Uint256]uint32)
	lastNotifySideMiningHeightMap = make(map[common.Uint256]uint32)
	lastSubmitAuxpowHeightMap = make(map[common.Uint256]uint32)
}
