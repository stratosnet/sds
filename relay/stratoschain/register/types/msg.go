package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
)

// ensure Msg interface compliance at compile time
var (
	_ sdk.Msg = &MsgCreateResourceNode{}
	_ sdk.Msg = &MsgRemoveResourceNode{}
	_ sdk.Msg = &MsgCreateIndexingNode{}
	_ sdk.Msg = &MsgRemoveIndexingNode{}
)

type MsgCreateResourceNode struct {
	NetworkAddress string         `json:"network_address" yaml:"network_address"`
	PubKey         crypto.PubKey  `json:"pubkey" yaml:"pubkey"`
	Value          sdk.Coin       `json:"value" yaml:"value"`
	OwnerAddress   sdk.AccAddress `json:"owner_address" yaml:"owner_address"`
	Description    Description    `json:"description" yaml:"description"`
}

// NewMsgCreateResourceNode NewMsg<Action> creates a new Msg<Action> instance
func NewMsgCreateResourceNode(networkAddr string, pubKey crypto.PubKey, value sdk.Coin, ownerAddr sdk.AccAddress, description Description,
) MsgCreateResourceNode {
	return MsgCreateResourceNode{
		NetworkAddress: networkAddr,
		PubKey:         pubKey,
		Value:          value,
		OwnerAddress:   ownerAddr,
		Description:    description,
	}
}

func (msg MsgCreateResourceNode) Route() string {
	return RouterKey
}

func (msg MsgCreateResourceNode) Type() string {
	return "create_resource_node"
}

// ValidateBasic validity check for the CreateResourceNode
func (msg MsgCreateResourceNode) ValidateBasic() error {
	if msg.NetworkAddress == "" {
		return ErrEmptyNetworkAddr
	}
	if msg.OwnerAddress.Empty() {
		return ErrEmptyOwnerAddr
	}
	if !msg.Value.IsPositive() {
		return ErrValueNegative
	}
	//if msg.Description == (Description{}) {
	//	return ErrEmptyDescription
	//}
	return nil
}

func (msg MsgCreateResourceNode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg MsgCreateResourceNode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.OwnerAddress}
}

type MsgCreateIndexingNode struct {
	NetworkAddress string         `json:"network_address" yaml:"network_address"`
	PubKey         crypto.PubKey  `json:"pubkey" yaml:"pubkey"`
	Value          sdk.Coin       `json:"value" yaml:"value"`
	OwnerAddress   sdk.AccAddress `json:"owner_address" yaml:"owner_address"`
	Description    Description    `json:"description" yaml:"description"`
}

// NewMsgCreateIndexingNode NewMsg<Action> creates a new Msg<Action> instance
func NewMsgCreateIndexingNode(networkAddr string, pubKey crypto.PubKey, value sdk.Coin, ownerAddr sdk.AccAddress, description Description,
) MsgCreateIndexingNode {
	return MsgCreateIndexingNode{
		NetworkAddress: networkAddr,
		PubKey:         pubKey,
		Value:          value,
		OwnerAddress:   ownerAddr,
		Description:    description,
	}
}

func (msg MsgCreateIndexingNode) Route() string {
	return RouterKey
}

func (msg MsgCreateIndexingNode) Type() string {
	return "create_indexing_node"
}

func (msg MsgCreateIndexingNode) ValidateBasic() error {
	if msg.NetworkAddress == "" {
		return ErrEmptyNetworkAddr
	}
	if msg.OwnerAddress.Empty() {
		return ErrEmptyOwnerAddr
	}
	if !msg.Value.IsPositive() {
		return ErrValueNegative
	}
	//if msg.Description == (Description{}) {
	//	return ErrEmptyDescription
	//}
	return nil
}

func (msg MsgCreateIndexingNode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg MsgCreateIndexingNode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.OwnerAddress}
}

// MsgRemoveResourceNode - struct for removing resource node
type MsgRemoveResourceNode struct {
	ResourceNodeAddress sdk.AccAddress `json:"resource_node_address" yaml:"resource_node_address"`
	OwnerAddress        sdk.AccAddress `json:"owner_address" yaml:"owner_address"`
}

// NewMsgRemoveResourceNode creates a new MsgRemoveResourceNode instance.
func NewMsgRemoveResourceNode(resourceNodeAddr sdk.AccAddress, ownerAddr sdk.AccAddress) MsgRemoveResourceNode {
	return MsgRemoveResourceNode{
		ResourceNodeAddress: resourceNodeAddr,
		OwnerAddress:        ownerAddr,
	}
}

// Route implements the sdk.Msg interface.
func (msg MsgRemoveResourceNode) Route() string { return RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgRemoveResourceNode) Type() string { return "remove_resource_node" }

// GetSigners implements the sdk.Msg interface.
func (msg MsgRemoveResourceNode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.OwnerAddress}
}

// GetSignBytes implements the sdk.Msg interface.
func (msg MsgRemoveResourceNode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgRemoveResourceNode) ValidateBasic() error {
	if msg.ResourceNodeAddress.Empty() {
		return ErrEmptyResourceNodeAddr
	}
	if msg.OwnerAddress.Empty() {
		return ErrEmptyOwnerAddr
	}
	return nil
}

// MsgRemoveIndexingNode - struct for removing indexing node
type MsgRemoveIndexingNode struct {
	IndexingNodeAddress sdk.AccAddress `json:"indexing_node_address" yaml:"indexing_node_address"`
	OwnerAddress        sdk.AccAddress `json:"owner_address" yaml:"owner_address"`
}

// NewMsgRemoveIndexingNode creates a new MsgRemoveIndexingNode instance.
func NewMsgRemoveIndexingNode(indexingNodeAddr sdk.AccAddress, ownerAddr sdk.AccAddress) MsgRemoveIndexingNode {
	return MsgRemoveIndexingNode{
		IndexingNodeAddress: indexingNodeAddr,
		OwnerAddress:        ownerAddr,
	}
}

// Route implements the sdk.Msg interface.
func (msg MsgRemoveIndexingNode) Route() string { return RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgRemoveIndexingNode) Type() string { return "remove_indexing_node" }

// GetSigners implements the sdk.Msg interface.
func (msg MsgRemoveIndexingNode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.OwnerAddress}
}

// GetSignBytes implements the sdk.Msg interface.
func (msg MsgRemoveIndexingNode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgRemoveIndexingNode) ValidateBasic() error {
	if msg.IndexingNodeAddress.Empty() {
		return ErrEmptyIndexingNodeAddr
	}
	if msg.OwnerAddress.Empty() {
		return ErrEmptyOwnerAddr
	}
	return nil
}
