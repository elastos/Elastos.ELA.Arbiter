package store

import "testing"

func TestDataStoreImpl_MiningRecordRelated(t *testing.T) {
	datastore, err := OpenDataStore()
	if err != nil {
		t.Error("Open database error.")
	}

	genesisBlockAddress := "testAddress"
	err = datastore.SetMiningRecord(genesisBlockAddress, 100, 101, 1)
	if err != nil {
		t.Error("Set mining record error.")
	}

	var mainHeight, sideHeight uint32
	var offset uint8
	ok, err := datastore.GetMiningRecord(genesisBlockAddress, &mainHeight, &sideHeight, &offset)
	if !ok || err != nil {
		t.Error("Get mining record error.")
	}
	if mainHeight != 100 {
		t.Error("Get main height error.")
	}
	if sideHeight != 101 {
		t.Error("Get side height error.")
	}
	if offset != 1 {
		t.Error("Get offset error.")
	}

	err = datastore.SetMiningRecord(genesisBlockAddress, 102, 103, 2)
	if err != nil {
		t.Error("Set mining record error.")
	}

	ok, err = datastore.GetMiningRecord(genesisBlockAddress, &mainHeight, &sideHeight, &offset)
	if !ok || err != nil {
		t.Error("Get mining record error.")
	}
	if mainHeight != 102 {
		t.Error("Get main height error.")
	}
	if sideHeight != 103 {
		t.Error("Get side height error.")
	}
	if offset != 2 {
		t.Error("Get offset error.")
	}

	datastore.ResetDataStore()
}
