package config

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func setup() {
	InitMockConfig()
}

func TestGetRpcConfig(t *testing.T) {

	if len(Parameters.SideNodeList) != 2 {
		t.Error("Wrong side nodes count.")
	}

	for _, node := range Parameters.SideNodeList {
		rpcConfig, ok := GetRpcConfig(node.GenesisBlockAddress)
		if !ok {
			t.Errorf("Can not find node by : [%s]", node.GenesisBlockAddress)
		}
		if *rpcConfig != *node.Rpc {
			t.Error("Found wrong config")
		}
	}

	rpcConfig, ok := GetRpcConfig("EeM7JrxNdi8MzgBfDExcAUTRXgH3vVHn54")
	if !ok {
		t.Errorf("Can not find node by : [%s]", "EeM7JrxNdi8MzgBfDExcAUTRXgH3vVHn54")
	}
	if rpcConfig.HttpJsonPort != 20038 || rpcConfig.IpAddress != "localhost" {
		t.Error("Found wrong config")
	}
}
