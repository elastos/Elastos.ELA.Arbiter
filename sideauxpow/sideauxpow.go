package sideauxpow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/password"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/wallet"

	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	ela "github.com/elastos/Elastos.ELA/core"
)

var KeystoreDict = map[string]string{
	"39fc8ba05b0064381e51afed65b4cf91bb8db60efebc38242e965d1b1fed0701": "keystore1.dat",
	"e1773a0e7af0cc3272fd271be9d5026c4b636c0b25cfe82be69d8bcc44ec512e": "keystore2.dat",
}

var (
	CurrentWallet wallet.Wallet
	Passwd        []byte
)

func getPassword(passwd []byte, confirmed bool) []byte {
	var tmp []byte
	var err error
	if len(passwd) > 0 {
		tmp = []byte(passwd)
	} else {
		if confirmed {
			tmp, err = password.GetConfirmedPassword()
		} else {
			tmp, err = password.GetPassword()
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	return tmp
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

func sideMiningTransfer(name string, passwd []byte, sideNode *config.SideNodeConfig) error {
	log.Info("getSideAuxpow")

	resp, err := rpc.CallAndUnmarshal("createauxblock", rpc.Param("paytoaddress", "EN1WeHcjgtkxrg1AoBNBdo3eY5fektuBZe"), sideNode.Rpc)
	if err != nil {
		return err
	}
	if resp == nil {
		log.Info("Create auxblock, nil ")
		return nil
	}

	type SideAuxBlock struct {
		GenesisHash       string `json:"genesishash"`
		Height            uint64 `json:"height"`
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		PreviousBlockHash string `json:"previousblockhash"`
	}

	sideAuxBlock := &SideAuxBlock{}

	err = unmarshal(resp, sideAuxBlock)
	if err != nil {
		return err
	}
	// fmt.Println(sideAuxBlock)

	txType := ela.SideMining

	sideGenesisHashData, _ := HexStringToBytes(sideAuxBlock.GenesisHash)
	sideBlockHashData, _ := HexStringToBytes(sideAuxBlock.Hash)

	sideGenesisHash, _ := Uint256FromBytes(sideGenesisHashData)
	sideBlockHash, _ := Uint256FromBytes(sideBlockHashData)

	fmt.Println(sideGenesisHash, sideBlockHash)
	// Create payload
	txPayload := &ela.PayloadSideMining{
		BlockHeight:     sideAuxBlock.Height,
		SideBlockHash:   *sideBlockHash,
		SideGenesisHash: *sideGenesisHash,
	}

	// create transaction
	feeStr := "0.001"

	fee, err := StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	keystore, err := wallet.OpenKeystore(name, Passwd)
	if err != nil {
		return err
	}

	from := keystore.Address()

	to := from
	amountStr := "0.1"
	amount, err := StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}

	genesisAddress, err := calculateGenesisAddress(sideAuxBlock.GenesisHash)
	if err != nil {
		return err
	}

	var txn *ela.Transaction
	txn, err = CurrentWallet.CreateAuxpowTransaction(txType, txPayload, from, to, genesisAddress, amount, fee)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	// sign transaction
	program := txn.Programs[0]

	haveSign, needSign, err := crypto.GetSignStatus(program.Code, program.Parameter)
	if haveSign == needSign {
		return errors.New("transaction was fully signed, no need more sign")
	}
	_, err = CurrentWallet.Sign(name, getPassword(passwd, false), txn)
	if err != nil {
		return err
	}
	haveSign, needSign, _ = crypto.GetSignStatus(program.Code, program.Parameter)
	log.Debug("Transaction successfully signed: ", haveSign, needSign)

	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	content := BytesToHexString(buf.Bytes())
	// log.Debug("Raw Sidemining transaction: ", content)

	// send transaction
	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}
	log.Debug("Send Sidemining transaction: ", result)

	return nil
}

func StartSideChainMining(sideNode *config.SideNodeConfig) {
	log.Info("[StartSideChainMining]Send side mining ")
	keystoreFile := KeystoreDict[sideNode.GenesisBlock]
	err := sideMiningTransfer(keystoreFile, Passwd, sideNode)
	if err != nil {
		log.Warn(err)
	}
}

func calculateGenesisAddress(genesisBlockHash string) (string, error) {
	genesisBlockBytes, err := HexStringToBytes(genesisBlockHash)
	if err != nil {
		return "", errors.New("genesis block hash to bytes failed")
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(byte(len(genesisBlockBytes)))
	buf.Write(genesisBlockBytes)
	buf.WriteByte(byte(CROSSCHAIN))

	genesisProgramHash, err := crypto.ToProgramHash(buf.Bytes())
	if err != nil {
		return "", errors.New("genesis block bytes to program hash faild")
	}

	genesisAddress, err := genesisProgramHash.ToAddress()
	if err != nil {
		return "", errors.New("genesis block hash to genesis address failed")
	}

	return genesisAddress, nil
}

func TestMultiSidechain() {
	for {
		select {
		case <-time.After(time.Second * 3):
			for _, node := range config.Parameters.SideNodeList {
				StartSideChainMining(node)
			}
			println("TestMultiSidechain")
		}
	}
}
