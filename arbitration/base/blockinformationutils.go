package base

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	types "github.com/elastos/Elastos.ELA/core/types/common"
	elatx "github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

var SystemAssetId = getSystemAssetId()

func getSystemAssetId() common.Uint256 {

	tx := elatx.CreateTransaction(
		types.TxVersion09,
		types.RegisterAsset,
		payload.TransferCrossChainVersionV1,
		&payload.RegisterAsset{
			Asset: payload.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: common.Uint168{},
		},
		[]*types.Attribute{},
		[]*types.Input{},
		[]*types.Output{},
		0,
		[]*program.Program{},
	)

	return tx.Hash()
}
