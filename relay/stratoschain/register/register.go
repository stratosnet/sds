package register

import (
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stratosnet/sds/relay/stratoschain/register/types"
	"github.com/stratosnet/sds/utils/crypto/secp256k1"
	utiltypes "github.com/stratosnet/sds/utils/types"
)

func BuildCreateResourceNodeMsg(networkAddress, token, moniker string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) (sdktypes.Msg, error) {
	tmPubkey, err := secp256k1.PubKeyBytesToTendermint(pubKey)
	if err != nil {
		return nil, err
	}
	return types.NewMsgCreateResourceNode(
		networkAddress,
		tmPubkey,
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		types.Description{
			Moniker: moniker,
		},
	), nil
}

func BuildCreateIndexingNodeMsg(networkAddress, token, moniker string, pubKey []byte, amount int64, ownerAddress utiltypes.Address) (sdktypes.Msg, error) {
	tmPubkey, err := secp256k1.PubKeyBytesToTendermint(pubKey)
	if err != nil {
		return nil, err
	}

	return types.NewMsgCreateIndexingNode(
		networkAddress,
		tmPubkey,
		sdktypes.NewInt64Coin(token, amount),
		ownerAddress[:],
		types.Description{
			Moniker: moniker,
		},
	), nil
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
