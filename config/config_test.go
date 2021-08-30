package config

import (
	"encoding/json"
	"io/ioutil"
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

type Salary struct {
	Basic, HRA, TA float64
}

type Employee struct {
	FirstName, LastName, Email string
	Age                        int
	MonthlySalary              []Salary
}

func TestJsonIndent(t *testing.T) {
	data := Employee{
		FirstName: "Mark",
		LastName:  "Jones",
		Email:     "mark@gmail.com",
		Age:       25,
		MonthlySalary: []Salary{
			Salary{
				Basic: 15000.00,
				HRA:   5000.00,
				TA:    2000.00,
			},
			Salary{
				Basic: 16000.00,
				HRA:   5000.00,
				TA:    2100.00,
			},
			Salary{
				Basic: 17000.00,
				HRA:   5000.00,
				TA:    2200.00,
			},
		},
	}
	file, _ := json.MarshalIndent(data, "", " ")
	_ = ioutil.WriteFile("test.json", file, 0644)

	data.MonthlySalary = append(data.MonthlySalary, Salary{
		Basic: 170000.00,
		HRA:   50000.00,
		TA:    22000.00,
	})

	file, _ = json.MarshalIndent(data, "", " ")
	_ = ioutil.WriteFile("test.json", file, 0644)

}
