package examples

import (
	"fmt"
	"os"
	"testing"

	"github.com/elastos/Elastos.ELA.Arbiter/common/config"
)

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func setup() {
	config.InitMockConfig()
}

func ExampleSimpleTest() {
	fmt.Println("1")
	//Output: 1
}
