package event

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
	"google.golang.org/protobuf/proto"
)

func verifyRspUploadFile(msg *protos.RspUploadFile) error {
	if msg == nil {
		return errors.New("RspUploadFile msg is empty")
	}
	if msg.SpP2PAddress == "" || msg.NodeSign == nil {
		return errors.New("key information is missing")
	}
	spP2pPubkey, err := requests.GetSpPubkey(msg.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp's pubkey, ")
	}
	if !types.VerifyP2pAddrBytes(spP2pPubkey, msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}

	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	signmsg, err := utils.GetRspUploadFileSpNodeSignMessage(msg)
	if err != nil {
		return errors.New("failed getting sp's sign message")
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// RspUploadFileVerifier task level verifier for all messages carrying RspUploadFile in a uploading task
func RspUploadFileVerifier(ctx context.Context, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// RspUploadFile itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.RspUploadFile{}) {
		return verifyRspUploadFile(target.(*protos.RspUploadFile))
	} else {
		// other types carrying a RspUploadFile
		field := reflect.ValueOf(target).Elem().FieldByName("RspUploadFile")
		if field.IsValid() {
			return verifyRspUploadFile(field.Interface().(*protos.RspUploadFile))
		} else {
			return errors.New("field of RspUploadFile is not found in the message")
		}
	}
}

func verifyRspBackupStatus(msg *protos.RspBackupStatus) error {
	if msg == nil {
		return errors.New("RspBackupStatus msg is empty")
	}
	if msg.SpP2PAddress == "" || msg.NodeSign == nil {
		return errors.New("key information is missing")
	}
	spP2pPubkey, err := requests.GetSpPubkey(msg.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp's pubkey, ")
	}
	if !types.VerifyP2pAddrBytes(spP2pPubkey, msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}
	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	signmsg, err := utils.GetRspBackupFileSpNodeSignMessage(msg)
	if err != nil {
		return errors.New("failed getting sp's sign message")
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// RspBackupStatusVerifier task level verifier for all messages carrying RspBackupStatus in a backup task
func RspBackupStatusVerifier(ctx context.Context, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// RspUploadFile itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.RspBackupStatus{}) {
		return verifyRspBackupStatus(target.(*protos.RspBackupStatus))
	} else {
		// other types carrying a RspUploadFile
		field := reflect.ValueOf(target).Elem().FieldByName("RspBackupFile")
		if field.IsValid() {
			return verifyRspBackupStatus(field.Interface().(*protos.RspBackupStatus))
		} else {
			return errors.New("field of RspUploadFile is not found in the message")
		}
	}
}

func verifyReqFileSliceBackupNotice(msg *protos.ReqFileSliceBackupNotice) error {
	if msg == nil {
		return errors.New("ReqFileSliceBackupNotice msg is empty")
	}
	if msg.SpP2PAddress == "" || msg.NodeSign == nil {
		return errors.New("key information is missing")
	}
	spP2pPubkey, err := requests.GetSpPubkey(msg.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp's pubkey, ")
	}
	if !types.VerifyP2pAddrBytes(spP2pPubkey, msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}
	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	signmsg, err := utils.GetReqBackupSliceNoticeSpNodeSignMessage(msg)
	if err != nil {
		return errors.New("failed getting sp's sign message")
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// ReqFileSliceBackupNoticeVerifier task level verifier for all messages carrying ReqFileSliceBackupNotice in a transfer task
func ReqFileSliceBackupNoticeVerifier(ctx context.Context, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// RspUploadFile itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.ReqFileSliceBackupNotice{}) {
		return verifyReqFileSliceBackupNotice(target.(*protos.ReqFileSliceBackupNotice))
	} else {
		// other types carrying a RspUploadFile
		field := reflect.ValueOf(target).Elem().FieldByName("RspFileStorageInfo")
		if field.IsValid() {
			return verifyReqFileSliceBackupNotice(field.Interface().(*protos.ReqFileSliceBackupNotice))
		} else {
			return errors.New("field of RspUploadFile is not found in the message")
		}
	}
}

func verifyRspFileStorageInfo(msg *protos.RspFileStorageInfo) error {
	if msg == nil {
		return errors.New("RspFileStorageInfo msg is empty")
	}
	if msg.SpP2PAddress == "" || msg.NodeSign == nil {
		return errors.New("key information is missing")
	}
	spP2pPubkey, err := requests.GetSpPubkey(msg.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp's pubkey, ")
	}
	if !types.VerifyP2pAddrBytes(spP2pPubkey, msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}
	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	signmsg, err := utils.GetRspFileStorageInfoNodeSignMessage(msg)
	if err != nil {
		return errors.New("failed getting sp's sign message")
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// RspFileStorageInfoVerifier task level verifier for all messages carrying RspFileStorageInfo in a download task
func RspFileStorageInfoVerifier(ctx context.Context, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// RspUploadFile itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.RspFileStorageInfo{}) {
		return verifyRspFileStorageInfo(target.(*protos.RspFileStorageInfo))
	} else {
		// other types carrying a RspUploadFile
		field := reflect.ValueOf(target).Elem().FieldByName("RspFileStorageInfo")
		if field.IsValid() {
			return verifyRspFileStorageInfo(field.Interface().(*protos.RspFileStorageInfo))
		} else {
			return errors.New("field of RspUploadFile is not found in the message")
		}
	}
}
