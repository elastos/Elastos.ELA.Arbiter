package examples

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/mainchain"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/sidechain"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/stretchr/testify/assert"
)

func init() {
	config.InitMockConfig()
	client, _ := account.Open("./wallet.dat", []byte("123"))
	arbitrator.Init(client)
	log.Init("./log", 1, 32, 64)
	testing.Init()
}

type TestMainChainFunc struct {
	//suite.Suite
}

func (mcFunc *TestMainChainFunc) GetWithdrawUTXOsByAmount(withdrawBank string, fixed64 common.Fixed64) ([]*store.AddressUTXO, error) {
	return mcFunc.GetAvailableUtxos()
}

func (mcFunc *TestMainChainFunc) GetAmountByInputs(inputs []*types.Input) (common.Fixed64, error) {
	return common.Fixed64(10000000000), nil
}

func (mcFunc *TestMainChainFunc) GetAvailableUtxos() ([]*store.AddressUTXO, error) {
	var utxos []*store.AddressUTXO
	amount := common.Fixed64(10000000000)
	utxo := &store.AddressUTXO{
		Input: &types.Input{
			Previous: types.OutPoint{
				TxID:  common.Uint256{},
				Index: 0,
			},
			Sequence: 0,
		},
		Amount:              &amount,
		GenesisBlockAddress: "XQd1DCi6H62NQdWZQhJCRnrPn7sF9CTjaU",
	}
	utxos = append(utxos, utxo)
	return utxos, nil
}

func (mcFunc *TestMainChainFunc) GetMainNodeCurrentHeight() (uint32, error) {
	return 200, nil
}

//This example demonstrate normal procedure of deposit
//As we known, the entire procedure will involve main chain, side chain, client of main chain
//	and client of side chain. To simplify this, we suppose the others are running well, and
//	we already known the result of these procedures.
func TestNormalWithdraw(t *testing.T) {

	//--------------Part1(On client of side chain)-------------------------
	//Step1.1 create transaction(tx3)
	//	./ela-cli wallet -t create --withdraw EeM7JrxNdi8MzgBfDExcAUTRXgH3vVHn54 --amount 1 --fee 0.1

	//Step1.2 sign tx3
	//	./ela-cli wallet -t sign --hex

	//Step1.3 send tx3 to side chain

	//--------------Part2(On side chain)-------------------------
	//Step2.1 tx3 has been confirmed and packaged in a new block

	//--------------Part3(On arbiter)-------------------------
	//let's suppose we get the object of current on duty arbitrator
	arbitrator := arbitrator.ArbitratorImpl{}
	mc := &mainchain.MainChainImpl{
		&cs.DistributedNodeServer{},
	}
	arbitrator.SetMainChain(mc)
	ArbitratorGroupSingleton.InitArbitratorsByStrings(
		[]string{
			"03a5274a21aa242231a1a95f88d1508be31a782303becaedc99f0016c46d105d7f",
			"03b8fbf8aa1eba7b7ccb7b4925a56ea71e487ea6fe0ec9c3ff0c725d3850a7b34f",
		},
		0,
	)

	//step3.1 SideChainAccountMonitorImpl found tx3 and fire utxo changed event

	//step3.2 SideChainImpl (correspond to side chain on part2) is a listener of SideChainAccountMonitorImpl,
	// the method OnUTXOChanged will be called automatically

	//let's suppose we already have the SideChainImpl object(from SideChainManager of the arbiter)
	sidechainObj := &sidechain.SideChainImpl{
		Key: "XQd1DCi6H62NQdWZQhJCRnrPn7sF9CTjaU",
	}

	//let's suppose we already known the serialized string of tx3 to simulate the object passed
	// from parameter of OnUTXOChanged
	var strTx3 string
	strTx3 = "08000122456261506d65774a63584575555733446138464a68506665754a386956557565483100e00f97000000000001001335353737303036373931393437373739343130059a160631fd3b332d97685bbde279ae0795aa8f7afd6ec2fe56ff21238615e7330100000000002cfd5e8457827ddc5051844d0326fe290fec27cb7d920a7133b3c36fb58fb9d60100fefffffff9eb8f672a8f8555103c376ecf7a86421400a4b835e9dc9d82e7003d5428b8fc0100feffffff300676308144d58dc6177b17e7e747a570e7a8dba0be3c57486e85104d68312e0100feffffff23fc2b13fa5aba5edc43f36daa513787f036e020eaf39c6fadee1cc9aaa6ef300100feffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3809698000000000000000000000000000000000000000000000000000000000000b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3fdb62300000000000000000021291454167350dc6e64059a34358225be84bd817100000000010023210271c405c657b59502547e45d86d1b49f5278b2d67431493c631a405fde7bec13cac"
	var tx3 *types.Transaction

	tx3 = new(types.Transaction)
	byteTx1, err := common.HexStringToBytes(strTx3)
	txReader := bytes.NewReader(byteTx1)
	tx3.Deserialize(txReader)
	strTx3 = common.BytesToHexString(byteTx1)
	//fmt.Printf("Withdraw transaction: [%s]", strTx3)

	//step3.3 parse withdraw info from tx3
	//withdrawInfos, _ := sidechainObj.parseUserWithdrawTransactionInfo([]*types.Transaction{tx3})

	var withdrawTxs []*base.WithdrawTx

	//hash := tx3.Hash()
	//step3.4 create withdraw transactions(tx4) for main chain
	tx4s := arbitrator.CreateWithdrawTransaction(withdrawTxs, sidechainObj, &TestMainChainFunc{})
	bufTx4 := new(bytes.Buffer)
	tx4s.Serialize(bufTx4)
	strTx4 := common.BytesToHexString(bufTx4.Bytes())

	//step3.5 broadcast withdraw proposal(contains tx4) into p2p network of arbitrators for collecting sign
	// from other arbitrators
	//arbitrator.BroadcastWithdrawProposal(tx4s)

	// note: details of broadcasting and sign collecting will be demonstrated in p2p_sign_collecting_test

	//step3.6 collecting enough sign then send tx4 to main chain

	//let's suppose we already known the serialized string of tx4 which contains complete signs

	//get from p2p_sign_collecting_test result

	//arbitrator.SendWithdrawTransaction(signedTx4)

	//--------------Part4(On main chain)-------------------------
	//step4.1 main chain node received tx4

	//step4.2 special verify of tx4

	//step4.3 tx4 has been confirmed and packaged in a new block
	fmt.Printf("Unsigned withdraw transaction: [%s]", strTx4)
	assert.NoError(t, err)
}
