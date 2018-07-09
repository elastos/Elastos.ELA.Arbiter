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
		rpcConfig, ok := GetRpcConfig(node.GenesisBlock)
		if !ok {
			t.Errorf("Can not find node by : [%s]", node.GenesisBlock)
		}
		if *rpcConfig != *node.Rpc {
			t.Error("Found wrong config")
		}
	}

	rpcConfig, ok := GetRpcConfig("168db7dedf19f584cd9acfc6062bb04a92ad1b7d34aed69905d4361728761a7c")
	if !ok {
		t.Errorf("Can not find node by : [%s]", "168db7dedf19f584cd9acfc6062bb04a92ad1b7d34aed69905d4361728761a7c")
	}
	if rpcConfig.HttpJsonPort != 20038 || rpcConfig.IpAddress != "localhost" {
		t.Error("Found wrong config")
	}
}
