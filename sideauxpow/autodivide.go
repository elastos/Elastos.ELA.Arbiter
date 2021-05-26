package sideauxpow

import (
	"bytes"
	"errors"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

type SideChainPowAccount struct {
	Address          string
	availableBalance common.Fixed64
}

func checkSideChainPowAccounts(addresses []string, minThreshold int) ([]*SideChainPowAccount, error) {
	var warnAddresses []*SideChainPowAccount
	currentHeight := arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()
	for _, addr := range addresses {
		available := common.Fixed64(0)
		locked := common.Fixed64(0)
		programHash, _ := common.Uint168FromAddress(addr)
		UTXOs, err := GetAddressUTXOs(programHash)
		if err != nil {
			return nil, errors.New("get " + addr + " UTXOs failed")
		}
		for _, utxo := range UTXOs {
			if utxo.LockTime < currentHeight {
				available += *utxo.Amount
			} else {
				locked += *utxo.Amount
			}
		}

		if available < common.Fixed64(minThreshold) {
			warnAddresses = append(warnAddresses, &SideChainPowAccount{
				Address:          addr,
				availableBalance: available,
			})
		}
	}

	if len(warnAddresses) > 0 {
		var warningStr string
		for _, sideChainPowAccount := range warnAddresses {
			warningStr += sideChainPowAccount.Address
			warningStr += ": "
			warningStr += sideChainPowAccount.availableBalance.String()
			warningStr += " "
		}

		log.Info("Warning side chain mining account: ", warningStr)

		return warnAddresses, nil
	}

	return nil, nil
}

func divideTransfer(name string, outputs []*Transfer) error {
	// create transaction
	fee := common.Fixed64(100000)
	mainAccount:= client.GetMainAccount()

	from := mainAccount.Address
	script := mainAccount.RedeemScript

	txType := types.TransferAsset
	txPayload := &payload.TransferAsset{}
	txn, err := createTransaction(txType, txPayload, from, &fee, script,
		uint32(0), arbitrator.ArbitratorGroupSingleton.GetCurrentHeight(), outputs...)
	if err != nil {
		return errors.New("create divide transaction failed: " + err.Error())
	}

	txnSigned, err := client.Sign(txn)
	if err != nil {
		return err
	}
	program := txnSigned.Programs[0]
	haveSign, needSign, _ := crypto.GetSignStatus(program.Code, program.Parameter)
	log.Debug("Divide transaction successfully signed: ", haveSign, needSign)

	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	content := common.BytesToHexString(buf.Bytes())

	// send transaction
	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}
	log.Debug("Send divide transaction: ", result)

	return nil
}

func SidechainAccountDivide() {
	for {
		select {
		case <-time.After(time.Second * 60):
			miningAddresses := make([]string, 0)
			for _, sideNode := range config.Parameters.SideNodeList {
				miningAddresses = append(miningAddresses, sideNode.MiningAddr)
			}
			warningAccounts, err := checkSideChainPowAccounts(miningAddresses, config.Parameters.MinThreshold)
			if err != nil {
				log.Error("Check side chain pow err", err)
			}
			if len(warningAccounts) > 0 {
				var outputs []*Transfer
				amount := common.Fixed64(config.Parameters.DepositAmount)
				for _, warningAccount := range warningAccounts {
					outputs = append(outputs, &Transfer{
						Address: warningAccount.Address,
						Amount:  &amount,
					})
				}
				divideTransfer(config.Parameters.WalletPath, outputs)
			}
		}
	}
}
