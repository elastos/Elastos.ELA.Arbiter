package main

import (
	"io"
	"os"
	"path/filepath"
	"net/http"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/password"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/cs"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/mainchain"
	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/sidechain"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/net/servers/httpjsonrpc"
	"github.com/elastos/Elastos.ELA.Arbiter/sideauxpow"
	"github.com/elastos/Elastos.ELA.Arbiter/store"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA.SPV/interface"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/utils/elalog"

)

var (
	LogsPath             = filepath.Join(config.DataPath, config.LogDir)
	ArbiterLogOutputPath = filepath.Join(LogsPath, "arbiter")
	SpvLogOutputPath     = filepath.Join(LogsPath, "spv")
)

const (
	defaultSpvMaxPerLogFileSize int64 = elalog.MBSize * 20
	defaultSpvMaxLogsFolderSize int64 = elalog.GBSize * 2

	defaultArbiterMaxPerLogFileSize int64 = 20
	defaultArbiterMaxLogsFolderSize int64 = 2 * 1024
)

func init() {
	spvMaxPerLogFileSize := defaultSpvMaxPerLogFileSize
	spvMaxLogsFolderSize := defaultSpvMaxLogsFolderSize
	if config.Parameters.MaxPerLogSize > 0 {
		spvMaxPerLogFileSize = int64(config.Parameters.MaxPerLogSize) * elalog.MBSize
	}
	if config.Parameters.MaxLogsSize > 0 {
		spvMaxLogsFolderSize = int64(config.Parameters.MaxLogsSize) * elalog.MBSize
	}
	fileWriter := elalog.NewFileWriter(
		SpvLogOutputPath,
		spvMaxPerLogFileSize,
		spvMaxLogsFolderSize,
	)
	logWriter := io.MultiWriter(os.Stdout, fileWriter)
	level := elalog.Level(config.Parameters.SPVPrintLevel)
	backend := elalog.NewBackend(logWriter, elalog.Llongfile)

	spvslog := backend.Logger("SPVS", level)
	_interface.UseLogger(spvslog)

	arbiterMaxPerLogFileSize := defaultArbiterMaxPerLogFileSize
	arbiterMaxLogsFolderSize := defaultArbiterMaxLogsFolderSize
	if config.Parameters.MaxPerLogSize > 0 {
		arbiterMaxPerLogFileSize = int64(config.Parameters.MaxPerLogSize)
	}
	if config.Parameters.MaxLogsSize > 0 {
		arbiterMaxLogsFolderSize = int64(config.Parameters.MaxLogsSize)
	}

	log.Init(
		ArbiterLogOutputPath,
		config.Parameters.PrintLevel,
		arbiterMaxPerLogFileSize,
		arbiterMaxLogsFolderSize,
	)

	log.Info("Init wallet.")
	passwd, err := password.GetAccountPassword()
	if err != nil {
		log.Fatal("Get password error.")
		os.Exit(1)
	}
	c, err := account.Open(sideauxpow.DefaultKeystoreFile, passwd)
	if err != nil || c == nil {
		log.Fatal("error: open wallet failed, ", err)
		os.Exit(1)
	}

	sideauxpow.Init(c)
	arbitrator.Init(c)
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
	pk, err := arbitrator.GetPublicKey().EncodePoint(true)
	if err != nil {
		return err
	}

	var id peer.PID
	copy(id[:], pk)
	if err := cs.InitP2PClient(id); err != nil {
		return err
	}

	//register p2p client listener
	if err := mainchain.InitMainChain(arbitrator); err != nil {
		return err
	}
	for _, side := range arbitrator.GetSideChainManager().GetAllChains() {
		cs.P2PClientSingleton.AddSidechainListener(side)
	}

	cs.P2PClientSingleton.Start()
	return nil
}

func main() {
	log.Info("Arbiter version: ", config.Version)

	log.Info("1. Init chain utxo cache.")
	dataStore, err := store.OpenDataStore()
	if err != nil {
		log.Fatalf("Data store open failed error: [s%]", err.Error())
		os.Exit(1)
	}
	store.DbCache = *dataStore

	log.Info("2. Init finished transaction cache.")
	finishedDataStore, err := store.OpenFinishedTxsDataStore()
	if err != nil {
		log.Fatalf("Side chain monitor setup error: [s%]", err.Error())
		os.Exit(1)
	}
	store.FinishedTxsDbCache = finishedDataStore

	currentArbitrator := arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator()

	log.Info("3. Start arbitrator P2P networks.")
	if err := initP2P(currentArbitrator); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	setSideChainAccountMonitor(currentArbitrator)

	log.Info("4. Init configurations.")
	if err := arbitrator.ArbitratorGroupSingleton.InitArbitrators(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("5. Start arbitrator spv module.")
	if err := currentArbitrator.StartSpvModule(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Info("6. Start arbitrator group monitor.")
	go arbitrator.ArbitratorGroupSingleton.SyncLoop()

	log.Info("7. Start servers.")
	pServer := new(http.Server)
	go httpjsonrpc.StartRPCServer(pServer)

	log.Info("8. Start check and remove cross chain transactions from db.")
	go currentArbitrator.CheckAndRemoveCrossChainTransactionsFromDBLoop()

	log.Info("9. Start side chain account divide.")
	go sideauxpow.SidechainAccountDivide()

	select {}
}
