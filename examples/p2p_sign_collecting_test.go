package examples

import (
	"bytes"
	"fmt"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"

	. "github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA/common"
	. "github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/crypto"
)

func init() {
	config.InitMockConfig()
	arbitrator.Init()
	log.Init("logs_test", 1, 20, 200)
}

type TestDistrubutedItemFunc struct {
}

func (tf *TestDistrubutedItemFunc) GetArbitratorGroupInfoByHeight(height uint32) (*rpc.ArbitratorGroupInfo, error) {
	return &rpc.ArbitratorGroupInfo{
		OnDutyArbitratorIndex: 0,
		Arbitrators: []string{
			"03a5274a21aa242231a1a95f88d1508be31a782303becaedc99f0016c46d105d7f",
			"03b8fbf8aa1eba7b7ccb7b4925a56ea71e487ea6fe0ec9c3ff0c725d3850a7b34f",
		},
	}, nil
}

//Collecting sign mostly used in withdraw procedure, which happens in the arbitrators' p2p network.
// Due to we can't collecting sign by order like normal multi-sign offline, the sign process is
// distributed and the current on duty arbitrator will extract the sign data and rearrange it.
func ExampleSignCollectingOfTwoArbitrators() {

	//We will simulate the collecting procedure without the p2p network transforming by letting
	// the two arbitrators within the same process.

	//get keystore string from keystore.dat
	/*dataStore, err := store.OpenDataStore()
	if err != nil {
		log.Fatalf("Side chain monitor setup error: [s%]", err.Error())
		os.Exit(1)
	}
	store.DbCache = dataStore

	onDutyArbitrator := &arbitrator.ArbitratorImpl{}
	testOnDutyArbitrator := &arbitrator.ArbitratorImpl{}

	onDutyArbitrator.InitAccount()
	testOnDutyArbitrator.Keystore = NewKeystore()

	onDutyKestoreStr, _ := onDutyArbitrator.Keystore.Json()
	fmt.Println(onDutyKestoreStr)

	strPassword := "123"
	testOnDutyArbitrator.Keystore.FromJson(onDutyKestoreStr, strPassword)
	testOnDutyKestoreStr, _ := testOnDutyArbitrator.Keystore.Json()
	fmt.Println(testOnDutyKestoreStr)

	if onDutyKestoreStr == testOnDutyKestoreStr {
		fmt.Println("OK\n")
	}*/

	onDutyArbitrator := &arbitrator.ArbitratorImpl{}
	anotherArbitrator := &arbitrator.ArbitratorImpl{}

	onDutyArbitrator.Keystore = NewKeystore()
	anotherArbitrator.Keystore = NewKeystore()

	onDutyKestoreStr := "{\"Version\":\"1.0\",\"IV\":\"cd96b862bc12fa10b3350def64601e77\",\"PasswordHash\":\"3180b4071170db0ae9f666167ed379f53468463f152e3c3cfb57d1de45fd01d6\",\"MasterKeyEncrypted\":\"8ce30a71cbc6e2d2a2a37ea7e7e2b3615accbe4cfe0e4212c6124d665863a455\",\"PrivateKeyEncrypted\":\"c9e66e5a0b8531e2bf3244358ecd226686230c71e76bbfa490c88f291ce604137dd1a117e24711b0f735c232d1d572fbb48663feab357fc1f1dc88cab62ed402d0ec2a4e579ff774f40b0ead26c9c48a234e9e4461e7321bd8ab60428bcaeeca\",\"SubAccountsCount\":0}"
	anotherKeystoreStr := "{\"Version\":\"1.0\",\"IV\":\"29931941e8929e02399267be04cbfb85\",\"PasswordHash\":\"3180b4071170db0ae9f666167ed379f53468463f152e3c3cfb57d1de45fd01d6\",\"MasterKeyEncrypted\":\"dbba23fca3421f5444337479986b06d55de9a618d417c06421cb49e4a25c5893\",\"PrivateKeyEncrypted\":\"b2a737bb753281e995a341400e723b999ddd8ce99e6f9583d98ffdc2910befba9b43218a997d8dec4feb91080e35eee726f9172ca9d1ee2c5550b5e2b16b8f79bf77b614ad7b9478a82f15e7e5f8d6da6ac40cf4bc61c14ccc9c9443a42394bd\",\"SubAccountsCount\":0}"
	onDutyKestorePassword := "123"
	anotherKestorePassword := "123"

	onDutyArbitrator.Keystore.FromJson(onDutyKestoreStr, onDutyKestorePassword)
	anotherArbitrator.Keystore.FromJson(anotherKeystoreStr, anotherKestorePassword)

	//let's suppose we already have a withdraw transaction(like tx4 referenced in withdraw_procedure_test)
	strTx4 := "0700c80000002258516431444369364836324e5164575a51684a43526e72506e3773463943546a6155012f8b43ceeb8b0754f401389d556bcac3e2d907d8af32c0407b9ee13754d5fbac0100133535373730303637393139343737373934313001000000000000000000000000000000000000000000000000000000000000000000000000000002b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3e00f9700000000000000000021ca13da099f035e055107850fafe241ec040c8920b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3804d735302000000000000004b9194e833a95201b915d8c55b18c54a2bb7248cd800000000010047522103a5274a21aa242231a1a95f88d1508be31a782303becaedc99f0016c46d105d7f2103b8fbf8aa1eba7b7ccb7b4925a56ea71e487ea6fe0ec9c3ff0c725d3850a7b34f52af"

	var tx4 *Transaction
	tx4 = new(Transaction)
	byteTx1, _ := common.HexStringToBytes(strTx4)
	txReader := bytes.NewReader(byteTx1)
	tx4.Deserialize(txReader)

	//--------------Part1(On duty arbitrator sending)-------------------------
	//note: the whole process happens on BroadcastWithdrawProposal method of DistributedNodeServer

	//step1.1 generate distributed item object for transforming and collecting
	programHash, _ := StandardAcccountPublicKeyToProgramHash(anotherArbitrator.GetPublicKey())
	transactionItem := &cs.DistributedItem{
		ItemContent:                 tx4,
		TargetArbitratorPublicKey:   anotherArbitrator.GetPublicKey(),
		TargetArbitratorProgramHash: programHash,
	}

	//step1.2 init redeem script for multi-sign(procedure of )
	publicKeys := make([]*crypto.PublicKey, 2)
	publicKeys[0] = onDutyArbitrator.GetPublicKey()
	publicKeys[1] = anotherArbitrator.GetPublicKey()
	redeemScript, _ := CreateWithdrawRedeemScript(2, publicKeys)
	transactionItem.SetRedeemScript(redeemScript)

	//step1.3 on duty arbitrator sign she self
	transactionItem.Sign(onDutyArbitrator, false, &TestDistrubutedItemFunc{})

	//step1.4 reset target and send to another arbitrator
	transactionItem.TargetArbitratorPublicKey = onDutyArbitrator.GetPublicKey()
	programHashOnDuty, _ := StandardAcccountPublicKeyToProgramHash(onDutyArbitrator.GetPublicKey())
	transactionItem.TargetArbitratorProgramHash = programHashOnDuty

	buf := new(bytes.Buffer)
	transactionItem.Serialize(buf)
	proposal := buf.Bytes()

	//step1.5 init tx4 programs after serialize distributed item object
	tx4.Programs[0].Parameter = transactionItem.GetSignedData()

	//--------------Part2(Another arbitrator)-------------------------
	//note: the whole process happens on OnP2PReceived method of DistributedNodeClient

	//step2.1 deserialize the proposal
	transactionItem2 := &cs.DistributedItem{}
	transactionItem2.Deserialize(bytes.NewReader(proposal))

	//step2.2 another arbitrator sign the proposal
	transactionItem2.Sign(anotherArbitrator, true, &TestDistrubutedItemFunc{})

	//step2.3 reset item target and send back
	transactionItem2.TargetArbitratorPublicKey = anotherArbitrator.GetPublicKey()
	programHash2, _ := StandardAcccountPublicKeyToProgramHash(anotherArbitrator.GetPublicKey())
	transactionItem2.TargetArbitratorProgramHash = programHash2

	buf2 := new(bytes.Buffer)
	transactionItem2.Serialize(buf2)
	proposal2 := buf2.Bytes()

	//--------------Part2(On duty arbitrator receiving)-------------------------
	//note: the whole process happens on OnP2PReceived method of DistributedNodeServer

	//step3.1 deserialize the proposal
	transactionItem3 := &cs.DistributedItem{}
	transactionItem3.Deserialize(bytes.NewReader(proposal2))

	//step3.2 parse sign data of another arbitrator
	newSign, _ := transactionItem3.ParseFeedbackSignedData()

	//step3.3 merge new sign
	num, _ := MergeSignToTransaction(newSign, 1, tx4)

	//finally we have an valid transaction(tx4) that can be sent to main node
	bufSignedTx4 := new(bytes.Buffer)
	tx4.Serialize(bufSignedTx4)
	strSignedTx4 := common.BytesToHexString(bufSignedTx4.Bytes())
	log.Info("strSignedTx4:", strSignedTx4)

	fmt.Printf("Number of signature is: [%d]\n", num)

	//Output:
	// Number of signature is: [2]
}
