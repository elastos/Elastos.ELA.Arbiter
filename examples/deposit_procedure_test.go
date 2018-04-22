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
	"github.com/elastos/Elastos.ELA.Utility/bloom"
	"github.com/elastos/Elastos.ELA.Utility/common"
	. "github.com/elastos/Elastos.ELA.Utility/core"
)

func init() {
	config.InitMockConfig()
	log.Init(log.Path, log.Stdout)
}

//This example demonstrate normal procedure of deposit
//As we known, the entire procedure will involve main chain, side chain, client of main chain
//	and client of side chain. To simplify this, we suppose the others are running well, and
//	we already known the result of these procedures.
func ExampleNormalDeposit() {

	//--------------Part1(On client of main chain)-------------------------
	//Step1.1 create transaction(tx1)
	//	./ela-cli wallet -t create -deposit ESsmDv6wZzHGFoHtN5Yqe31GqSzJnuaVWZ --amount 30 --fee 1

	//Step1.2 sign tx1
	//	./ela-cli wallet -t sign --hex

	//Step1.3 send tx1 to main chain

	//--------------Part2(On main chain)-------------------------
	//Step2.1 tx1 has been confirmed and packaged in a new block

	//--------------Part3(On arbiter)-------------------------
	//let's suppose we get the object of current on duty arbitrator
	arbitrator := arbitrator.ArbitratorImpl{}
	mc := &mainchain.MainChainImpl{&cs.DistributedNodeServer{P2pCommand: cs.WithdrawCommand}}
	arbitrator.SetMainChain(mc)

	sideChainManager := &sidechain.SideChainManagerImpl{make(map[string]SideChain)}
	side := &sidechain.SideChainImpl{
		nil,
		"EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y",
		nil,
	}
	sideChainManager.AddChain("EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y", side)
	arbitrator.SetSideChainManager(sideChainManager)

	//Step3.1 spv module found the tx1, and fire Notify callback of TransactionListener

	//let's suppose we already known the serialized code of tx1 and proof of tx1 from the callback
	var strTx1 string
	strTx1 = "0800012245544d4751433561473131627752677553704357324e6b7950387a75544833486e3200010013353537373030363739313934373737393431300403229feeff99fa03357d09648a93363d1d01f234e61d04d10f93c9ad1aef3c150100feffffff737a4387ebf5315b74c508e40ba4f0179fc1d68bf76ce079b6bbf26e0fd2aa470100feffffff592c415c08ac1e1312d98cf6a28f68b62dd28ae964ed33af882b2d16b3a44a900100feffffff34255723e2249e8d965892edb9cd4cbbe27fa30e1292372a07206079dfad4a260100feffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b00000000000000002132a3f3d36f0db243743debee55155d5343322c2ab037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3782e43120000000000000000216fd749255076c304942d16a8023a63b504b6022f570200000100232103c3ffe56a4c68b4dfe91573081898cb9a01830e48b8f181de684e415ecfc0e098ac"
	strProof := ""

	var tx1 *Transaction
	tx1 = new(Transaction)
	byteTx1, _ := common.HexStringToBytes(strTx1)
	txReader := bytes.NewReader(byteTx1)
	tx1.Deserialize(txReader)

	var proof bloom.MerkleProof
	byteProof, _ := common.HexStringToBytes(strProof)
	proofReader := bytes.NewReader(byteProof)
	proof.Deserialize(proofReader)

	//step3.2 parse deposit info from tx1
	depositInfos, _ := arbitrator.ParseUserDepositTransactionInfo(tx1)

	//step3.3 create transaction(tx2) info from deposit info
	transactionInfos := arbitrator.CreateDepositTransactions(proof, depositInfos)

	//step3.4 send tx2 info to side chain
	//arbitrator.SendDepositTransactions(transactionInfos)

	//--------------Part4(On side chain)-------------------------
	//step4.1 side chain node received tx2 info

	//let's suppose we already known the serialized tx2 info
	var serializedTx2 string
	for info := range transactionInfos {
		infoDataReader := new(bytes.Buffer)
		info.Serialize(infoDataReader)
		serializedTx2 = common.BytesToHexString(infoDataReader.Bytes())
	}

	//convert tx2 info to tx2

	//step4.2 special verify of tx2 which contains spv proof of tx1

	//step4.3 tx2 has been confirmed and packaged in a new block

	fmt.Printf("Length of transaction info array: [%d]\n"+
		"Serialized tx2: [%s]",
		len(transactionInfos), serializedTx2)

	// Output:
	// Length of transaction info array: [1]
	// Serialized tx2: [06005a3030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030300100133535373730303637393139343737373934313000000140623033376462393634613233313435386432643666666435656131383934346334663930653633643534376335643362393837346466363661346561643061330231302245544d4751433561473131627752677553704357324e6b7950387a75544833486e32000000000000000001000000000000000000000000000000000000]
}
