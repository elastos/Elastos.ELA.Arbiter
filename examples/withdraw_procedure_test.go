package examples

import (
	"bytes"
	"fmt"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/mainchain"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/sidechain"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA/core"
)

func init() {
	config.InitMockConfig()
	arbitrator.Init()
	log.Init(log.Path, log.Stdout)
}

type TestMainChainFunc struct {
}

func (mcFunc *TestMainChainFunc) GetAvailableUtxos(withdrawBank string) ([]*store.AddressUTXO, error) {
	var utxos []*store.AddressUTXO
	amount := common.Fixed64(10000000000)
	utxo := &store.AddressUTXO{
		Input: &Input{
			Previous: OutPoint{
				TxID:  common.Uint256{},
				Index: 0,
			},
			Sequence: 0,
		},
		Amount:              &amount,
		GenesisBlockAddress: "EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y",
		DestroyAddress:      "EeM7JrxNdi8MzgBfDExcAUTRXgH3vVHn54",
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
func ExampleNormalWithdraw() {

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
		&cs.DistributedNodeServer{P2pCommand: cs.WithdrawCommand},
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
		Key: "EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y",
	}

	//let's suppose we already known the serialized string of tx3 to simulate the object passed
	// from parameter of OnUTXOChanged
	var strTx3 string
	strTx3 = "080001224555646452564e4e624e5a544836727954546b4631706a3250637054704e4b356d720058020000000000000100133535373730303637393139343737373934313001550aca5407374432c10c375b4a59948982938426795a54f625104fa95466bf8c01000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3bc020000000000000000000021e879146ef5119f34ce35b2f50624deea68c74924b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a340ce00000000000000000000210981c4d4e838dc9a7b3892bafca10097a88ba60800000000010023210269ec2f545f1658ff9a3e38ea7b5bd8c1d4aaebb9e69a358a8e2acbfdc4a05094ac"
	var tx3 *Transaction

	tx3 = new(Transaction)
	byteTx1, _ := common.HexStringToBytes(strTx3)
	txReader := bytes.NewReader(byteTx1)
	tx3.Deserialize(txReader)

	//step3.3 parse withdraw info from tx3
	withdrawInfos, _ := sidechainObj.ParseUserWithdrawTransactionInfo(tx3)

	hash := tx3.Hash()
	//step3.4 create withdraw transactions(tx4) for main chain
	tx4s := arbitrator.CreateWithdrawTransactions(withdrawInfos, sidechainObj, hash.String(), &TestMainChainFunc{})
	bufTx4 := new(bytes.Buffer)
	tx4s[0].Serialize(bufTx4)
	strTx4 := common.BytesToHexString(bufTx4.Bytes())

	//step3.5 broadcast withdraw proposal(contains tx4) into p2p network of arbitrators for collecting sign
	// from other arbitrators
	// arbitrator.BroadcastWithdrawProposal(tx4s)

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

	// Output:
	// Unsigned withdraw transaction: [0700c800000022454d6d66676e72444c516d4650424a6957767379594756326a7a4c515935384a345940333464386433303938393538633565306432323166343336653362393034323637326237626131353434333764643434643866616536363933323130626132370100133535373730303637393139343737373934313001000000000000000000000000000000000000000000000000000000000000000000000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3580200000000000006000000217de79d9145c3684752f1b78699e1aac3f00a6787b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a344e10b5402000000000000002132a3f3d36f0db243743debee55155d5343322c2a00000000010047522103a5274a21aa242231a1a95f88d1508be31a782303becaedc99f0016c46d105d7f2103b8fbf8aa1eba7b7ccb7b4925a56ea71e487ea6fe0ec9c3ff0c725d3850a7b34f52af]
}
