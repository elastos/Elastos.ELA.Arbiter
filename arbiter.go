package main

import (
	"os"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/mainchain"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/sidechain"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
	"github.com/elastos/Elastos.ELA.Arbiter/password"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.Arbiter/store"
	"github.com/elastos/Elastos.ELA.Arbiter/wallet"
	"github.com/elastos/Elastos.ELA.Utility/elalog"
	//"github.com/elastos/Elastos.ELA.Utility/p2p/addrmgr"
	//"github.com/elastos/Elastos.ELA.Utility/p2p/connmgr"
	"github.com/elastos/Elastos.ELA.Utility/p2p/peer"
)

var (
	backend = elalog.NewBackend(log.Stdout, elalog.Llongfile)

	addrlog = backend.Logger("ADDR", 0)
	connlog = backend.Logger("CONN", 0)
	peerlog = backend.Logger("PEER", 0)
)

func init() {
	config.Init()
	log.Init(log.Path, log.Stdout)

	//addrmgr.UseLogger(addrlog)
	//connmgr.UseLogger(connlog)
	peer.UseLogger(peerlog)

	arbitrator.Init()
	sidechain.Init()
}

func setSideChainAccountMonitor(arbitrator arbitrator.Arbitrator) {
	monitor := sidechain.SideChainAccountMonitorImpl{ParentArbitrator: arbitrator}

	for _, side := range arbitrator.GetSideChainManager().GetAllChains() {
		monitor.AddListener(side)
	}

	for _, node := range config.Parameters.SideNodeList {
		go monitor.SyncChainData(node)
	}
}

func initP2P(arbitrator arbitrator.Arbitrator) error {
	if err := cs.InitP2PClient(arbitrator); err != nil {
		return err
	}

	//register p2p client listener
	if err := mainchain.InitMainChain(arbitrator); err != nil {
		return err
	}
	for _, side := range arbitrator.GetSideChainManager().GetAllChains() {
		cs.P2PClientSingleton.AddListener(side)
	}

	cs.P2PClientSingleton.Start()
	return nil
}

func main() {
	log.Info("Arbiter version: ", config.Version)
	log.Info("1. Init configurations.")
	if err := arbitrator.ArbitratorGroupSingleton.InitArbitrators(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("2. Init chain utxo cache.")
	dataStore, err := store.OpenDataStore()
	if err != nil {
		log.Fatalf("Data store open failed error: [s%]", err.Error())
		os.Exit(1)
	}
	store.DbCache = *dataStore

	log.Info("3. Init finished transaction cache.")
	finishedDataStore, err := store.OpenFinishedTxsDataStore()
	if err != nil {
		log.Fatalf("Side chain monitor setup error: [s%]", err.Error())
		os.Exit(1)
	}
	store.FinishedTxsDbCache = finishedDataStore

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()

	log.Info("4. Init wallet.")
	passwd, err := password.GetAccountPassword()
	if err != nil {
		log.Fatal("Get password error.")
		os.Exit(1)
	}
	wallet, err := wallet.Open(passwd)
	if err != nil {
		log.Fatal("error: open wallet failed, ", err)
		os.Exit(1)
	}
	sideauxpow.CurrentWallet = wallet

	log.Info("5. Init arbitrator account.")
	sideauxpow.SetMainAccountPassword(passwd)
	if err := currentArbitrator.InitAccount(passwd); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("6. Start arbitrator P2P networks.")
	if err := initP2P(currentArbitrator); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	setSideChainAccountMonitor(currentArbitrator)
	currentArbitrator.GetMainChain().SyncChainData()
	currentArbitrator.GetArbitratorGroup().CheckOnDutyStatus()

	log.Info("7. Start arbitrator spv module.")
	if err := currentArbitrator.StartSpvModule(passwd); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("8. Start arbitrator group monitor.")
	go arbitrator.ArbitratorGroupSingleton.SyncLoop()

	log.Info("9. Start servers.")
	go httpjsonrpc.StartRPCServer()

	log.Info("10. Start check and remove cross chain transactions from db.")
	go currentArbitrator.CheckAndRemoveCrossChainTransactionsFromDBLoop()

	log.Info("11. Start side chain account divide.")
	go sideauxpow.SidechainAccountDivide(wallet)

	select {}
}
