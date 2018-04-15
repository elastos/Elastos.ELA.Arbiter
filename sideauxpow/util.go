package sideauxpow

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "github.com/elastos/Elastos.ELA.Arbiter/common"
	walt "github.com/elastos/Elastos.ELA.Arbiter/wallet"
)

func selectAddress(wallet walt.Wallet) (string, error) {
	addresses, err := wallet.GetAddresses()
	if err != nil || len(addresses) == 0 {
		return "", errors.New("fail to load wallet addresses")
	}

	// only one address return it
	if len(addresses) == 1 {
		return addresses[0].Address, nil
	}

	// print out addresses in wallet
	fmt.Println("INDEX ADDRESS"+strings.Repeat(" ", len(addresses[0].Address)-7), "BALANCE")
	fmt.Println("----- -------"+strings.Repeat("-", len(addresses[0].Address)-7), "--------------------")
	for i, address := range addresses {
		balance := Fixed64(0)
		utxos, err := wallet.GetAddressUTXOs(address.ProgramHash)
		if err != nil {
			return "", errors.New("get " + address.Address + " UTXOs failed")
		}
		for _, utxo := range utxos {
			balance += *utxo.Amount
		}
		fmt.Printf("%5d %s %s\n", i+1, address.Address, balance)
		fmt.Println("----- -------"+strings.Repeat("-", len(addresses[0].Address)-7), "--------------------")
	}

	// select address by index input
	fmt.Println("Please input the address INDEX you want to use and press enter")

	index := -1
	for index == -1 {
		index = getInput(len(addresses))
	}

	return addresses[index].Address, nil
}

func getInput(max int) int {
	fmt.Print("INPUT INDEX: ")
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Println("read input falied")
		return -1
	}

	// trim space
	input = strings.TrimSpace(input)

	index, err := strconv.ParseInt(input, 10, 32)
	if err != nil {
		fmt.Println("please input a positive integer")
		return -1
	}

	if int(index) > max {
		fmt.Println("INDEX should between 1 ~", max)
		return -1
	}

	return int(index) - 1
}
