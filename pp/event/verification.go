package event

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/stratosnet/sds/framework/core"
	fwcrypto "github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/msg/header"
	fwtypes "github.com/stratosnet/sds/framework/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/sds-msg/protos"
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
	if !fwtypes.VerifyP2pAddrBytes(spP2pPubkey.Bytes(), msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}

	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return errors.New("failed encoding sp's sign message")
	}
	msgSignBytes := fwcrypto.CalcHashBytes(msgBytes)
	if !fwtypes.VerifyP2pSignBytes(spP2pPubkey.Bytes(), nodeSign, msgSignBytes) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// RspUploadFileVerifier task level verifier for all messages carrying RspUploadFile in a uploading task
func RspUploadFileVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	err := verifyReqId(ctx, msgType.Id)
	if err != nil {
		return err
	}
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// RspUploadFile itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.RspUploadFile{}) {
		// from sp, verify the p2p address
		err := verifySpP2pAddress(ctx, msgType.Name)
		if err != nil {
			return err
		}
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

// RspUploadFileWithNoReqIdVerifier no reqid verification for a request message from gateway pp
func RspUploadFileWithNoReqIdVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// other types carrying a RspUploadFile
	field := reflect.ValueOf(target).Elem().FieldByName("RspUploadFile")
	if field.IsValid() {
		return verifyRspUploadFile(field.Interface().(*protos.RspUploadFile))
	} else {
		return errors.New("field of RspUploadFile is not found in the message")
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
	if !fwtypes.VerifyP2pAddrBytes(spP2pPubkey.Bytes(), msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}
	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return errors.New("failed to encoding sp's sign message")
	}
	msgSignBytes := fwcrypto.CalcHashBytes(msgBytes)
	if !fwtypes.VerifyP2pSignBytes(spP2pPubkey.Bytes(), nodeSign, msgSignBytes) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// RspBackupStatusVerifier task level verifier for all messages carrying RspBackupStatus in a backup task
func RspBackupStatusVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	err := verifyReqId(ctx, msgType.Id)
	if err != nil {
		return err
	}
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// RspBackupStatus itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.RspBackupStatus{}) {
		return verifyRspBackupStatus(target.(*protos.RspBackupStatus))
	} else {
		// other types carrying a RspBackupStatus
		field := reflect.ValueOf(target).Elem().FieldByName("RspBackupFile")
		if field.IsValid() {
			return verifyRspBackupStatus(field.Interface().(*protos.RspBackupStatus))
		} else {
			return errors.New("field of RspBackupFile is not found in the message")
		}
	}
}

// RspBackupStatusWithNoReqIdVerifier no reqid verification for a request message from gateway pp
func RspBackupStatusWithNoReqIdVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}

	// other types carrying a RspBackupStatus
	field := reflect.ValueOf(target).Elem().FieldByName("RspBackupFile")
	if field.IsValid() {
		return verifyRspBackupStatus(field.Interface().(*protos.RspBackupStatus))
	} else {
		return errors.New("field of RspBackupFile is not found in the message")
	}
}

func verifyNoticeFileSliceBackup(msg *protos.NoticeFileSliceBackup) error {
	if msg == nil {
		return errors.New("NoticeFileSliceBackup msg is empty")
	}
	if msg.SpP2PAddress == "" || msg.NodeSign == nil {
		return errors.New("key information is missing")
	}
	spP2pPubkey, err := requests.GetSpPubkey(msg.SpP2PAddress)
	if err != nil {
		return errors.Wrap(err, "failed to get sp's pubkey, ")
	}
	if !fwtypes.VerifyP2pAddrBytes(spP2pPubkey.Bytes(), msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}
	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return errors.New("failed to encoding sp's sign message")
	}
	signBytes := fwcrypto.CalcHashBytes(msgBytes)
	if !fwtypes.VerifyP2pSignBytes(spP2pPubkey.Bytes(), nodeSign, signBytes) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// NoticeFileSliceBackupVerifier task level verifier for all messages carrying NoticeFileSliceBackup in a transfer task
func NoticeFileSliceBackupVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// NoticeFileSliceBackup itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.NoticeFileSliceBackup{}) {
		return verifyNoticeFileSliceBackup(target.(*protos.NoticeFileSliceBackup))
	} else {
		// other types carrying a NoticeFileSliceBackup
		field := reflect.ValueOf(target).Elem().FieldByName("NoticeFileSliceBackup")
		if field.IsValid() {
			return verifyNoticeFileSliceBackup(field.Interface().(*protos.NoticeFileSliceBackup))
		} else {
			return errors.New("field of NoticeFileSliceBackup is not found in the message")
		}
	}
}

func verifyRspFileStorageInfo(msg *protos.RspFileStorageInfo) error {
	utils.DebugLog("verifyRspFileStorageInfo")
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
	if !fwtypes.VerifyP2pAddrBytes(spP2pPubkey.Bytes(), msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}
	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	msg.ReqId = ""

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return errors.New("failed to encoding sp's sign message")
	}
	msgSignBytes := fwcrypto.CalcHashBytes(msgBytes)

	if !fwtypes.VerifyP2pSignBytes(spP2pPubkey.Bytes(), nodeSign, msgSignBytes) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// RspFileStorageInfoVerifier task level verifier for all messages carrying RspFileStorageInfo in a download task
func RspFileStorageInfoVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	err := verifyReqId(ctx, msgType.Id)
	if err != nil {
		return err
	}
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}

	// RspUploadFile itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.RspFileStorageInfo{}) {
		// from sp, verify the p2p address
		err := verifySpP2pAddress(ctx, msgType.Name)
		if err != nil {
			return err
		}
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

// RspFileStorageInfoWithNoReqIdVerifier no reqid verification for a request message from gateway pp
func RspFileStorageInfoWithNoReqIdVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}

	// other types carrying a RspUploadFile
	field := reflect.ValueOf(target).Elem().FieldByName("RspFileStorageInfo")
	if field.IsValid() {
		return verifyRspFileStorageInfo(field.Interface().(*protos.RspFileStorageInfo))
	} else {
		return errors.New("field of RspFileStorageInfo is not found in the message")
	}
}
func verifySpP2pAddress(ctx context.Context, cmd string) error {
	p2pAddress := core.GetSrcP2pAddrFromContext(ctx)
	if _, ok := setting.SPMap.Load(p2pAddress); !ok {
		return errors.New(fmt.Sprintf("Source p2p address(%s) in a (%s) type message is not in the SP list", p2pAddress, cmd))
	}
	return nil
}

func verifyReqId(ctx context.Context, msgTypeId uint8) error {
	_, found := p2pserver.GetP2pServer(ctx).LoadRequestInfo(requests.GetReqIdFromMessage(ctx), msgTypeId)
	if !found {
		return errors.New("no previous request for this rsp found")
	}
	return nil
}

func PpRspVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	return verifyReqId(ctx, msgType.Id)
}

func SpRspVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	if err := verifySpP2pAddress(ctx, msgType.Name); err != nil {
		return err
	}
	return verifyReqId(ctx, msgType.Id)
}

func SpAddressVerifier(ctx context.Context, msgType header.MsgType, target interface{}) error {
	return verifySpP2pAddress(ctx, msgType.Name)
}
