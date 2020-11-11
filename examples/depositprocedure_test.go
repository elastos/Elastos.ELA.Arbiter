package examples

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/mainchain"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/sidechain"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/elanet/bloom"
)

func init() {
	config.InitMockConfig()
	log.Init("./log", 1, 32, 64)
}

//This example demonstrate normal procedure of deposit
//As we known, the entire procedure will involve main chain, side chain, client of main chain
//	and client of side chain. To simplify this, we suppose the others are running well, and
//	we already known the result of these procedures.
func TestNormalDeposit(t *testing.T) {

	//--------------Part1(On client of main chain)-------------------------
	//Step1.1 create transaction(tx1)
	//	./ela-cli wallet -t create -deposit EMmfgnrDLQmFPBJiWvsyYGV2jzLQY58J4Y --amount 10 --fee 1

	//Step1.2 sign tx1
	//	./ela-cli wallet -t sign --hex

	//Step1.3 send tx1 to main chain

	//--------------Part2(On main chain)-------------------------
	//Step2.1 tx1 has been confirmed and packaged in a new block

	//--------------Part3(On arbiter)-------------------------
	//let's suppose we get the object of current on duty arbitrator
	arbitrator := arbitrator.ArbitratorImpl{}
	mc := &mainchain.MainChainImpl{&cs.DistributedNodeServer{}}
	arbitrator.SetMainChain(mc)

	sideChainManager := &sidechain.SideChainManagerImpl{make(map[string]SideChain)}
	side := &sidechain.SideChainImpl{
		Key: "XQd1DCi6H62NQdWZQhJCRnrPn7sF9CTjaU",
	}
	sideChainManager.AddChain("XQd1DCi6H62NQdWZQhJCRnrPn7sF9CTjaU", side)
	arbitrator.SetSideChainManager(sideChainManager)

	//Step3.1 spv module found the tx1, and fire Notify callback of TransactionListener

	//let's suppose we already known the serialized code of tx1 and proof of tx1 from the callback
	var strTx1 string
	strTx1 = "08000122456261506d65774a63584575555733446138464a68506665754a386956557565483100e00f97000000000001001335353737303036373931393437373739343130059a160631fd3b332d97685bbde279ae0795aa8f7afd6ec2fe56ff21238615e7330100000000002cfd5e8457827ddc5051844d0326fe290fec27cb7d920a7133b3c36fb58fb9d60100fefffffff9eb8f672a8f8555103c376ecf7a86421400a4b835e9dc9d82e7003d5428b8fc0100feffffff300676308144d58dc6177b17e7e747a570e7a8dba0be3c57486e85104d68312e0100feffffff23fc2b13fa5aba5edc43f36daa513787f036e020eaf39c6fadee1cc9aaa6ef300100feffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38096980000000000000000004b9194e833a95201b915d8c55b18c54a2bb7248cd8b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3fdb62300000000000000000021291454167350dc6e64059a34358225be84bd817100000000010023210271c405c657b59502547e45d86d1b49f5278b2d67431493c631a405fde7bec13cac"
	strProof := "5f894325400c9a12f4490da7bca9f4e32466f497a65aacb2dbfa29ac14619944b300000001000000010000005f894325400c9a12f4490da7bca9f4e32466f497a65aacb2dbfa29ac14619944fd83010800012245544d4751433561473131627752677553704357324e6b7950387a75544833486e3200010013353537373030363739313934373737393431300403229feeff99fa03357d09648a93363d1d01f234e61d04d10f93c9ad1aef3c150100feffffff737a4387ebf5315b74c508e40ba4f0179fc1d68bf76ce079b6bbf26e0fd2aa470100feffffff592c415c08ac1e1312d98cf6a28f68b62dd28ae964ed33af882b2d16b3a44a900100feffffff34255723e2249e8d965892edb9cd4cbbe27fa30e1292372a07206079dfad4a260100feffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b00000000000000002132a3f3d36f0db243743debee55155d5343322c2ab037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3782e43120000000000000000216fd749255076c304942d16a8023a63b504b6022f570200000100232103c3ffe56a4c68b4dfe91573081898cb9a01830e48b8f181de684e415ecfc0e098ac"

	var tx1 *types.Transaction
	tx1 = new(types.Transaction)
	byteTx1, _ := common.HexStringToBytes(strTx1)
	txReader := bytes.NewReader(byteTx1)
	err := tx1.Deserialize(txReader)

	var proof bloom.MerkleProof
	byteProof, _ := common.HexStringToBytes(strProof)
	proofReader := bytes.NewReader(byteProof)
	proof.Deserialize(proofReader)

	//step3.2 parse deposit info from tx1
	//depositInfo, _ := arbitrator.ParseUserDepositTransactionInfo(tx1, side.GetKey())

	//step3.3 create transaction(tx2) info from deposit info
	//transactionInfos := arbitrator.CreateDepositTransactions([]*base.SpvTransaction{&base.SpvTransaction{tx1, &proof, depositInfo}})

	//step3.4 send tx2 info to side chain
	//arbitrator.SendDepositTransactions(transactionInfos)

	//--------------Part4(On side chain)-------------------------
	//step4.1 side chain node received tx2 info

	//let's suppose we already known the serialized tx2 info

	//convert tx2 info to tx2

	//step4.2 special verify of tx2 which contains spv proof of tx1

	//step4.3 tx2 has been confirmed and packaged in a new block

	assert.NoError(t, err)
}
