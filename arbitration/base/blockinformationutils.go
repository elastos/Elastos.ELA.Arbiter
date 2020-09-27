package base

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

var SystemAssetId = getSystemAssetId()

func getSystemAssetId() common.Uint256 {
	systemToken := &types.Transaction{
		TxType:         types.RegisterAsset,
		PayloadVersion: 0,
		Payload: &payload.RegisterAsset{
			Asset: payload.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: common.Uint168{},
		},
		Attributes: []*types.Attribute{},
		Inputs:     []*types.Input{},
		Outputs:    []*types.Output{},
		Programs:   []*program.Program{},
	}
	return systemToken.Hash()
}
