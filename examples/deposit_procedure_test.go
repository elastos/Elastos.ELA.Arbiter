package examples

import (
	"fmt"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	tx "github.com/elastos/Elastos.ELA.Arbiter/core/transaction"
	spv "github.com/elastos/Elastos.ELA.SPV/interface"
)

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
	//Step3.1 spv module found the tx1, and fire Notify callback of TransactionListener

	//let's suppose we already known the serialized code of tx1 and proof of tx1
	//strTx1 := "080001001335353737303036373931393437373739343130097a8a95e1b1500619ac8d580ddfed35e0af05ae517f8a84f68638206129696f7d0100feffffffb2f6277b253dda0b96dbe9ee7ccb10ad54c69f9f64af4cf1de9ad758d213afd50100feffffffb3662e1de29786ef8bd7f2186805d5be23131e37aeb57a2381f7f82d34c6569d0100feffffff175ee361d1709b890061fea045a2f60eb1a18c270523430c772dbcccc5b862a30100fefffffff6464235320ebb3807a53b452ff939af2d9f70296ab78a0791a25a83926826590100feffffff922f5eeb615bcf972118e812463e5e04074e548acf07db54a3f5d3381f3142a30100feffffff4d75fee59710aa59c135b5f2bebaa9084d2df16a6e92d769778a9dce59bb4bf20100feffffff81ef63eed85b2ec5ef66856330c135355c586b5a4e4dcce3aacd350e1cecb3c70100feffffff21e3eecc88f5b1d3ca5ad3014c5ac064094705082e5deb44d74349913e343ef70100feffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3005ed0b20000000000000000216aa3b2ea55c432c8833182922a8f22e1a43aa64fb037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a34e6ad6030000000000000000210bdb36abfc877cf36131911d42a651f0e2187995720000000141407b961028fa53710f0e1e059b8650e22e59187443870c3764adad12854171c752a539584a94690f8a31f4069f5835e5cde966f8ce1bdcb6bf072658b2d93e6781232102884d2487f0c98bb00b00d69c93ea6aa84daa4dc63ff272200947864a4220ba77ac"
	//strProof := ""
	var tx1 *tx.Transaction
	var proof spv.Proof

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

	//convert tx2 info to tx2

	//step4.2 special verify of tx2 which contains spv proof of tx1

	//step4.3 tx2 has been confirmed and packaged in a new block

	fmt.Printf("Length of transaction info array: [%t]\n, serialized tx2: [%s]",
		len(transactionInfos), serializedTx2)

	//Output:
	// Length of transaction info array: [1]
	// serialized tx2: []
}
