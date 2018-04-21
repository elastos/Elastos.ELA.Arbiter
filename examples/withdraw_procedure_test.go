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
	. "github.com/elastos/Elastos.ELA.Utility/core"
)

func init() {
	config.Init()
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
	//	./ela-cli wallet -t create --withdraw EZwPHEMQLNBpP2VStF3gRk8EVoMM2i3hda --amount 1 --fee 0.1

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
		nil,
		"EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y",
		nil,
	}

	//let's suppose we already known the serialized string of tx3 to simulate the object passed
	// from parameter of OnUTXOChanged
	var strTx3 string
	strTx3 = "0800012245544d4751433561473131627752677553704357324e6b7950387a75544833486e320001001335353737303036373931393437373739343130049147d096d23f3fa718ddcca4f0fc051f832f2b823020666aa16ccc65c03c4e3c0100feffffff737a4387ebf5315b74c508e40ba4f0179fc1d68bf76ce079b6bbf26e0fd2aa470100feffffff2593c8dc8e4d2106291ac6e77a298f75b598957f1e7efd0221ee76584d63abbe0100feffffff152186d284028bbff9b3ebcb016b0fab2088aa3c8105d77e1a23c6cc7de6856a0100feffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b000000000000000021e879146ef5119f34ce35b2f50624deea68c74924b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3782e43120000000000000000216fd749255076c304942d16a8023a63b504b6022f5d0300000100232103c3ffe56a4c68b4dfe91573081898cb9a01830e48b8f181de684e415ecfc0e098ac"
	var tx3 *Transaction

	tx3 = new(Transaction)
	byteTx1, _ := common.HexStringToBytes(strTx3)
	txReader := bytes.NewReader(byteTx1)
	tx3.Deserialize(txReader)

	//step3.3 parse withdraw info from tx3
	withdrawInfos, _ := sidechainObj.ParseUserWithdrawTransactionInfo(tx3)

	hash := tx3.Hash()
	//step3.4 create withdraw transactions(tx4) for main chain
	tx4s := arbitrator.CreateWithdrawTransaction(withdrawInfos, sidechainObj, hash.String(), &TestMainChainFunc{})
	bufTx4 := new(bytes.Buffer)
	tx4s[0].Serialize(bufTx4)
	strTx4 := common.BytesToHexString(bufTx4.Bytes())

	//step3.5 broadcast withdraw proposal(contains tx4) into p2p network of arbitrators for collecting sign
	// from other arbitrators
	// arbitrator.BroadcastWithdrawProposal(tx4s)

	// note: details of broadcasting and sign collecting will be demonstrated in p2p_sign_collecting_test

	//step3.6 collecting enough sign then send tx4 to main chain

	//let's suppose we already known the serialized string of tx4 which contains complete signs
	var strSignedTx4 string
	//get from p2p_sign_collecting_test result
	strSignedTx4 = "0700c800000022454d6d66676e72444c516d4650424a6957767379594756326a7a4c515935384a345940636335346461653936333865303937373833353239643635303630353239393833613435626465323163373635643438333939303964303135343930346539350100133535373730303637393139343737373934313001000000000000000000000000000000000000000000000000000000000000000000000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b0000000006000000216fd749255076c304942d16a8023a63b504b6022fb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3001a711802000000000000002132a3f3d36f0db243743debee55155d5343322c2a00000000018140ecba54c926d2cfe41e822d016db44e0e476c2c4f3cea9692b15a7ce78a03c8054c53c134dce62d2daa6e8755d482f99008e3bc4dfddf434227b4f1fec20f2872aba117872daa5e7cf77a2fb79117d19fa5bc05627f0747f36535254a97688b4d7b8d2a84becbb317ecf08bb9536c55e7085165c20a3c7142ef8ec2892b02347047522103a5274a21aa242231a1a95f88d1508be31a782303becaedc99f0016c46d105d7f2103b8fbf8aa1eba7b7ccb7b4925a56ea71e487ea6fe0ec9c3ff0c725d3850a7b34f52af"

	//arbitrator.SendWithdrawTransaction(signedTx4)

	//--------------Part4(On main chain)-------------------------
	//step4.1 main chain node received tx4

	//step4.2 special verify of tx4

	//step4.3 tx4 has been confirmed and packaged in a new block

	fmt.Printf("Unsigned withdraw transaction: [%s]\nmulti-signed withdraw transaction: [%s]", strTx4, strSignedTx4)

	//Output:
	// Unsigned withdraw transaction: [0700c800000022454d6d66676e72444c516d4650424a6957767379594756326a7a4c515935384a345940636335346461653936333865303937373833353239643635303630353239393833613435626465323163373635643438333939303964303135343930346539350100133535373730303637393139343737373934313001000000000000000000000000000000000000000000000000000000000000000000000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b0000000006000000216fd749255076c304942d16a8023a63b504b6022fb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3001a711802000000000000002132a3f3d36f0db243743debee55155d5343322c2a00000000010047522103a5274a21aa242231a1a95f88d1508be31a782303becaedc99f0016c46d105d7f2103b8fbf8aa1eba7b7ccb7b4925a56ea71e487ea6fe0ec9c3ff0c725d3850a7b34f52af]
	// multi-signed withdraw transaction: [0700c800000022454d6d66676e72444c516d4650424a6957767379594756326a7a4c515935384a345940636335346461653936333865303937373833353239643635303630353239393833613435626465323163373635643438333939303964303135343930346539350100133535373730303637393139343737373934313001000000000000000000000000000000000000000000000000000000000000000000000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b0000000006000000216fd749255076c304942d16a8023a63b504b6022fb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3001a711802000000000000002132a3f3d36f0db243743debee55155d5343322c2a00000000018140ecba54c926d2cfe41e822d016db44e0e476c2c4f3cea9692b15a7ce78a03c8054c53c134dce62d2daa6e8755d482f99008e3bc4dfddf434227b4f1fec20f2872aba117872daa5e7cf77a2fb79117d19fa5bc05627f0747f36535254a97688b4d7b8d2a84becbb317ecf08bb9536c55e7085165c20a3c7142ef8ec2892b02347047522103a5274a21aa242231a1a95f88d1508be31a782303becaedc99f0016c46d105d7f2103b8fbf8aa1eba7b7ccb7b4925a56ea71e487ea6fe0ec9c3ff0c725d3850a7b34f52af]
}
