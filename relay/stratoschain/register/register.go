package register

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/relay/stratoschain/register/types"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

func BuildCreateResourceNodeMsg(networkAddress, token, moniker string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) sdktypes.Msg {
	tmPubkey := secp256k1.PubKeyBytesToTendermint(pubKey)

	return types.NewMsgCreateResourceNode(
		networkAddress,
		tmPubkey,
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		types.Description{
			Moniker: moniker,
		},
	)
}

func BuildCreateIndexingNodeMsg(networkAddress, token, moniker string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) sdktypes.Msg {
	tmPubkey := secp256k1.PubKeyBytesToTendermint(pubKey)

	return types.NewMsgCreateIndexingNode(
		networkAddress,
		tmPubkey,
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		types.Description{
			Moniker: moniker,
		},
	)
}

func BuildRemoveResourceNodeMsg(nodeAddress, ownerAddress utiltypes.Address) sdktypes.Msg {
	return types.NewMsgRemoveResourceNode(
		nodeAddress[:],
		ownerAddress[:],
	)
}

func BuildRemoveIndexingNodeMsg(nodeAddress, ownerAddress utiltypes.Address) sdktypes.Msg {
	return types.NewMsgRemoveIndexingNode(
		nodeAddress[:],
		ownerAddress[:],
	)
}
