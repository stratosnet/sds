package sds

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/relay/stratoschain/sds/types"
)

func BuildFileUploadMsg(fileHash, reporterAddress, uploaderAddress []byte) (sdktypes.Msg, error) {
	return types.NewMsgUpload(
		fileHash,
		reporterAddress,
		uploaderAddress,
	), nil
}

func BuildPrepayMsg(token string, amount int64, senderAddress []byte) (sdktypes.Msg, error) {
	return types.NewMsgPrepay(
		senderAddress,
		sdktypes.NewCoins(sdktypes.NewInt64Coin(token, amount)),
	), nil
}
