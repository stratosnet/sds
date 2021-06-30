package types

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type FileInfo struct {
	Height   sdk.Int
	Reporter sdk.AccAddress
	Uploader sdk.AccAddress
}

// constructor
func NewFileInfo(height sdk.Int, reporter, uploader sdk.AccAddress) FileInfo {
	return FileInfo{
		Height:   height,
		Reporter: reporter,
		Uploader: uploader,
	}
}

// MustMarshalFileInfo returns the fileInfo's bytes. Panics if fails
func MustMarshalFileInfo(cdc *codec.Codec, file FileInfo) []byte {
	return cdc.MustMarshalBinaryLengthPrefixed(file)
}

// MustUnmarshalFileInfo unmarshal a file's info from a store value. Panics if fails
func MustUnmarshalFileInfo(cdc *codec.Codec, value []byte) FileInfo {
	file, err := UnmarshalFileInfo(cdc, value)
	if err != nil {
		panic(err)
	}
	return file
}

// UnmarshalResourceNode unmarshal a file's info from a store value
func UnmarshalFileInfo(cdc *codec.Codec, value []byte) (fi FileInfo, err error) {
	err = cdc.UnmarshalBinaryLengthPrefixed(value, &fi)
	return fi, err
}

// String returns a human readable string representation of a resource node.
func (fi FileInfo) String() string {
	return fmt.Sprintf(`FileInfo:{
		Height:				%s
  		Reporter:			%s
  		Uploader:			%s
	}`, fi.Height.String(), fi.Reporter.String(), fi.Uploader.String())
}
