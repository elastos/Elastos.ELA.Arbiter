package sideauxpow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/password"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/wallet"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA.Utility/core"
	. "github.com/elastos/Elastos.ELA.Utility/crypto"
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

func Transfer(name string, password []byte, wallet wallet.Wallet) error {

	fmt.Println("getSideAuxpow")
	resp, err := rpc.CallAndUnmarshal("createauxblock", rpc.Param("paytoaddress", "EN1WeHcjgtkxrg1AoBNBdo3eY5fektuBZe"), config.Parameters.SideNodeList[0].Rpc)
	if err != nil {
		return err
	}

	type SideAuxBlock struct {
		GenesisHash       string `json:"genesishash"`
		Height            uint64 `json:"height"`
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		PreviousBlockHash string `json:"previousblockhash"`
	}

	sideAuxBlock := &SideAuxBlock{}

	unmarshal(resp, sideAuxBlock)

	fmt.Println(sideAuxBlock)

	txType := SideMining

	sideGenesisHashData, _ := HexStringToBytes(sideAuxBlock.GenesisHash)
	sideBlockHashData, _ := HexStringToBytes(sideAuxBlock.Hash)

	sideGenesisHash, _ := Uint256FromBytes(sideGenesisHashData)
	sideBlockHash, _ := Uint256FromBytes(sideBlockHashData)
	// Create payload
	txPayload := &PayloadSideMining{
		SideBlockHash:   *sideBlockHash,
		SideGenesisHash: *sideGenesisHash,
	}

	// create transaction
	fee, err := StringToFixed64("0.01")
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	from := "EN1M19RYHuFPS91hNRzR15TNtoAUDhi7hk"
	if from == "" {
		from, err = selectAddress(wallet)
		if err != nil {
			return err
		}
	}

	to := from
	if to == "" {
		return errors.New("use --to to specify receiver address")
	}

	amountStr := "0.1"
	if amountStr == "" {
		return errors.New("use --amount to specify transfer amount")
	}

	amount, err := StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}

	lockStr := ""
	var txn *Transaction
	if lockStr == "" {
		txn, err = wallet.CreateTransaction(txType, txPayload, from, to, amount, fee)
		if err != nil {
			return errors.New("create transaction failed: " + err.Error())
		}
	} else {
		lock, err := strconv.ParseUint(lockStr, 10, 32)
		if err != nil {
			return errors.New("invalid lock height")
		}
		txn, err = wallet.CreateLockedTransaction(txType, txPayload, from, to, amount, fee, uint32(lock))
		if err != nil {
			return errors.New("create transaction failed: " + err.Error())
		}
	}

	// sign transaction
	haveSign, needSign, err := GetSignStatus(txn.Programs[0].Code, txn.Programs[0].Code)
	if haveSign == needSign {
		return errors.New("transaction was fully signed, no need more sign")
	}
	_, err = wallet.Sign(name, getPassword(password, false), txn)
	if err != nil {
		return err
	}
	haveSign, needSign, _ = GetSignStatus(txn.Programs[0].Code, txn.Programs[0].Parameter)
	fmt.Println("[", haveSign, "/", needSign, "] Transaction successfully signed")

	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	content := BytesToHexString(buf.Bytes())
	// Print transaction hex string content to console
	fmt.Println(content)

	// send transaction
	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("Data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}
	fmt.Println(result.(string))

	return nil
}
