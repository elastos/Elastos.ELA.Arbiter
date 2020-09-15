package base

import (
	"errors"
	"github.com/elastos/Elastos.ELA/common"
	"io"
)


type RegisteredSideChainTransaction struct {
	TransactionHash     string
	GenesisBlockAddress string
	RegisteredSideChain *RegisteredSideChain
}


type RegisteredSideChain struct {
	// Name of side chain
	SideChainName string

	// Magic number of side chain
	MagicNumber uint32

	// DNSSeeds defines a list of DNS seeds for the network to discover peers.
	DNSSeeds []string

	// Node port of side chain
	NodePort uint16

	// Genesis hash of side chain
	GenesisHash common.Uint256

	// Genesis block timestamp of side chain
	GenesisTimestamp uint32

	// Genesis block difficulty of side chain
	GenesisBlockDifficulty string

	// SideChain rpc port
	HttpJsonPort uint16

	// SideChain Ip Address
	IpAddr string

	// Username of rpc
	User string

	// Password of rpc
	Pass string
}


func (sc *RegisteredSideChain) Serialize(w io.Writer) error {
	if err := common.WriteVarString(w, sc.SideChainName); err != nil {
		return errors.New("fail to serialize SideChainName")
	}
	if err := common.WriteUint32(w, sc.MagicNumber); err != nil {
		return errors.New("fail to serialize MagicNumber")
	}

	if err := common.WriteVarUint(w, uint64(len(sc.DNSSeeds))); err != nil {
		return errors.New("failed to serialize DNSSeeds")
	}

	for _, v := range sc.DNSSeeds {
		if err := common.WriteVarString(w, v); err != nil {
			return errors.New("failed to serialize DNSSeeds")
		}
	}

	if err := common.WriteUint16(w, sc.NodePort); err != nil {
		return errors.New("failed to serialize NodePort")
	}

	if err := sc.GenesisHash.Serialize(w); err != nil {
		return errors.New("failed to serialize GenesisHash")
	}

	if err := common.WriteUint32(w, sc.GenesisTimestamp); err != nil {
		return errors.New("failed to serialize GenesisTimestamp")
	}

	if err := common.WriteVarString(w, sc.GenesisBlockDifficulty); err != nil {
		return errors.New("failed to serialize GenesisTimestamp")
	}

	if err := common.WriteUint16(w, sc.HttpJsonPort); err != nil {
		return errors.New("failed to serialize HttpJsonPort")
	}

	if err := common.WriteVarString(w, sc.IpAddr); err != nil {
		return errors.New("failed to serialize IpAddr")
	}

	if err := common.WriteVarString(w, sc.User); err != nil {
		return errors.New("failed to serialize User")
	}

	if err := common.WriteVarString(w, sc.Pass); err != nil {
		return errors.New("failed to serialize Pass")
	}
	return nil
}

func (sc *RegisteredSideChain) Deserialize(r io.Reader) error {
	var err error
	sc.SideChainName, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], SideChainName deserialize failed")
	}

	sc.MagicNumber, err = common.ReadUint32(r)
	if err != nil {
		return errors.New("[CRCProposal], MagicNumber deserialize failed")
	}

	length, err := common.ReadVarUint(r, 0)
	if err != nil {
		return errors.New("[CRCProposal], DNSSeeds length deserialize failed")
	}
	var seeds []string
	for i := 0; i < int(length); i++ {
		seed, err := common.ReadVarString(r)
		if err != nil {
			return errors.New("failed to deserialize DNSSeeds")
		}
		seeds = append(seeds, seed)
	}
	sc.DNSSeeds = seeds

	sc.NodePort, err = common.ReadUint16(r)
	if err != nil {
		return errors.New("[CRCProposal], NodePort deserialize failed")
	}

	if err := sc.GenesisHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize GenesisHash")
	}

	sc.GenesisTimestamp, err = common.ReadUint32(r)
	if err != nil {
		return errors.New("[CRCProposal], NodePort deserialize failed")
	}

	sc.GenesisBlockDifficulty, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], GenesisBlockDifficulty deserialize failed")
	}

	sc.HttpJsonPort, err = common.ReadUint16(r)
	if err != nil {
		return errors.New("[CRCProposal], HttpJsonPort deserialize failed")
	}

	sc.IpAddr, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], IpAddr deserialize failed")
	}

	sc.User, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], User deserialize failed")
	}

	sc.Pass, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Pass deserialize failed")
	}

	return nil
}


