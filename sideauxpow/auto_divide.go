package sideauxpow

import (
	"bytes"
	"errors"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	walt "github.com/elastos/Elastos.ELA.Arbiter/wallet"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	ela "github.com/elastos/Elastos.ELA/core"
)

type SideChainPowAccount struct {
	Address          string
	availableBalance Fixed64
}

func checkSideChainPowAccounts(addrs []*walt.Address, minThreshold int, wallet walt.Wallet) ([]*SideChainPowAccount, error) {
	var warnAddresses []*SideChainPowAccount
	currentHeight := wallet.CurrentHeight(walt.QueryHeightCode)
	for _, addr := range addrs {
		available := Fixed64(0)
		locked := Fixed64(0)
		UTXOs, err := wallet.GetAddressUTXOs(addr.ProgramHash)
		if err != nil {
			return nil, errors.New("get " + addr.Address + " UTXOs failed")
		}
		for _, utxo := range UTXOs {
			if utxo.LockTime < currentHeight {
				available += *utxo.Amount
			} else {
				locked += *utxo.Amount
			}
		}

		if available < Fixed64(minThreshold*100000000) {
			warnAddresses = append(warnAddresses, &SideChainPowAccount{
				Address:          addr.Address,
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

func divideTransfer(name string, passwd []byte, outputs []*walt.Transfer) error {
	// create transaction
	fee, err := StringToFixed64("0.001")
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	keystore, err := walt.OpenKeystore(name, getMainAccountPassword())
	if err != nil {
		return err
	}

	from := keystore.Address()

	var txn *ela.Transaction
	txn, err = CurrentWallet.CreateMultiOutputTransaction(from, fee, outputs...)
	if err != nil {
		return errors.New("create divide transaction failed: " + err.Error())
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
	log.Debug("Divide transaction successfully signed: ", haveSign, needSign)

	buf := new(bytes.Buffer)
	txn.Serialize(buf)
	content := BytesToHexString(buf.Bytes())

	// send transaction
	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return err
	}
	log.Debug("Send divide transaction: ", result)

	return nil
}

func SidechainAccountDivide(wallet walt.Wallet) {
	for {
		select {
		case <-time.After(time.Second * 3):
			addresses, err := wallet.GetAddresses()
			if err != nil {
				log.Error("Get addresses error:", err)
			}
			warningAccounts, err := checkSideChainPowAccounts(addresses, 10, wallet)
			if err != nil {
				log.Error("Check side chain pow err", err)
			}
			if len(warningAccounts) > 0 {
				var outputs []*walt.Transfer
				amount := Fixed64(10)
				for _, warningAccount := range warningAccounts {
					outputs = append(outputs, &walt.Transfer{
						Address: warningAccount.Address,
						Amount:  &amount,
					})
				}
				divideTransfer(walt.DefaultKeystoreFile, getMainAccountPassword(), outputs)
			}
		}
	}
}
