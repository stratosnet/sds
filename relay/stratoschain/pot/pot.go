package pot

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/relay/stratoschain/pot/types"
	"github.com/stratosnet/sds/sp/storages/table"
)

func BuildVolumeReportMsg(traffic []table.Traffic, reporterAddress []byte, epoch uint64, reportReference string) (sdktypes.Msg, error) {
	aggregatedVolume := make(map[string]uint64)
	for _, trafficReccord := range traffic {
		aggregatedVolume[trafficReccord.ProviderWalletAddress] += trafficReccord.Volume
	}

	var nodesVolume []types.SingleNodeVolume
	for address, volume := range aggregatedVolume {
		addressBytes, err := sdktypes.AccAddressFromBech32(address)
		if err != nil {
			return nil, err
		}
		nodesVolume = append(nodesVolume, types.SingleNodeVolume{
			NodeAddress: addressBytes,
			Volume:      sdktypes.NewIntFromUint64(volume),
		})
	}

	return types.NewMsgVolumeReport(nodesVolume, reporterAddress, sdktypes.NewIntFromUint64(epoch), reportReference), nil
}
