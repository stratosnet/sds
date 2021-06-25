package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const MsgType = "volume_report"

// verify interface at compile time
var (
	_ sdk.Msg = &MsgVolumeReport{}
	_ sdk.Msg = &MsgWithdraw{}
)

type MsgVolumeReport struct {
	NodesVolume     []SingleNodeVolume `json:"nodes_volume" yaml:"nodes_volume"`         // volume report
	Reporter        sdk.AccAddress     `json:"reporter" yaml:"reporter"`                 // volume reporter
	Epoch           sdk.Int            `json:"report_epoch" yaml:"report_epoch"`         // volume report epoch
	ReportReference string             `json:"report_reference" yaml:"report_reference"` // volume report reference
}

// NewMsgVolumeReport creates a new Msg<Action> instance
func NewMsgVolumeReport(
	nodesVolume []SingleNodeVolume,
	reporter sdk.AccAddress,
	epoch sdk.Int,
	reportReference string,
) MsgVolumeReport {
	return MsgVolumeReport{
		NodesVolume:     nodesVolume,
		Reporter:        reporter,
		Epoch:           epoch,
		ReportReference: reportReference,
	}
}

// Route Implement
func (msg MsgVolumeReport) Route() string { return RouterKey }

// GetSigners Implement
func (msg MsgVolumeReport) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Reporter}
}

// Type Implement
func (msg MsgVolumeReport) Type() string { return MsgType }

// GetSignBytes gets the bytes for the message signer to sign on
func (msg MsgVolumeReport) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic validity check for the AnteHandler
func (msg MsgVolumeReport) ValidateBasic() error {
	if msg.Reporter.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "missing reporter address")
	}
	if !(len(msg.NodesVolume) > 0) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "no node reports volume")
	}

	if !(msg.Epoch.IsPositive()) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid report epoch")
	}

	if !(len(msg.ReportReference) > 0) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid report reference hash")
	}

	for _, item := range msg.NodesVolume {
		if item.Volume.IsNegative() {
			return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "report volume is negative")
		}
		if item.NodeAddress.Empty() {
			return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "missing node address")
		}
	}
	return nil
}

type MsgWithdraw struct {
	Amount       sdk.Coin       `json:"amount" yaml:"amount"`
	NodeAddress  sdk.AccAddress `json:"node_address" yaml:"node_address"`
	OwnerAddress sdk.AccAddress `json:"owner_address" yaml:"owner_address"`
}

func NewMsgWithdraw(amount sdk.Coin, nodeAddress sdk.AccAddress, ownerAddress sdk.AccAddress) MsgWithdraw {
	return MsgWithdraw{
		Amount:       amount,
		NodeAddress:  nodeAddress,
		OwnerAddress: ownerAddress,
	}
}

// Route Implement
func (msg MsgWithdraw) Route() string { return RouterKey }

// GetSigners Implement
func (msg MsgWithdraw) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.OwnerAddress}
}

// Type Implement
func (msg MsgWithdraw) Type() string { return "withdraw" }

// GetSignBytes gets the bytes for the message signer to sign on
func (msg MsgWithdraw) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic validity check for the AnteHandler
func (msg MsgWithdraw) ValidateBasic() error {
	if !(msg.Amount.IsPositive()) {
		return ErrWithdrawAmountNotPositive
	}
	if msg.NodeAddress.Empty() {
		return ErrMissingNodeAddress
	}
	if msg.OwnerAddress.Empty() {
		return ErrMissingOwnerAddress
	}
	return nil
}
