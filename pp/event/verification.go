package event

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stratosnet/sds/framework/core"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/requests"
	"github.com/stratosnet/sds/pp/setting"
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
func RspUploadFileVerifier(ctx context.Context, cmd string, target interface{}) error {
	err := verifyReqId(ctx, cmd)
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
		err := verifySpP2pAddress(ctx, cmd)
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
func RspUploadFileWithNoReqIdVerifier(ctx context.Context, cmd string, target interface{}) error {
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
func RspBackupStatusVerifier(ctx context.Context, cmd string, target interface{}) error {
	err := verifyReqId(ctx, cmd)
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
func RspBackupStatusWithNoReqIdVerifier(ctx context.Context, cmd string, target interface{}) error {
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
func ReqFileSliceBackupNoticeVerifier(ctx context.Context, cmd string, target interface{}) error {
	msgBuf := core.MessageFromContext(ctx)
	if err := proto.Unmarshal(msgBuf.MSGBody, target.(proto.Message)); err != nil {
		return errors.Wrap(err, "protobuf Unmarshal error")
	}
	// ReqFileSliceBackupNotice itself
	if reflect.TypeOf(target) == reflect.TypeOf(&protos.ReqFileSliceBackupNotice{}) {
		return verifyReqFileSliceBackupNotice(target.(*protos.ReqFileSliceBackupNotice))
	} else {
		// other types carrying a ReqFileSliceBackupNotice
		field := reflect.ValueOf(target).Elem().FieldByName("ReqFileSliceBackupNotice")
		if field.IsValid() {
			return verifyReqFileSliceBackupNotice(field.Interface().(*protos.ReqFileSliceBackupNotice))
		} else {
			return errors.New("field of ReqFileSliceBackupNotice is not found in the message")
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
	if !types.VerifyP2pAddrBytes(spP2pPubkey, msg.SpP2PAddress) {
		return errors.Wrap(err, "failed to verify p2p address, ")
	}
	nodeSign := msg.NodeSign
	msg.NodeSign = nil
	msg.ReqId = ""
	signmsg, err := utils.GetRspFileStorageInfoNodeSignMessage(msg)
	utils.DebugLogf("file storage info signmsg: %v", msg)
	if err != nil {
		return errors.New("failed getting sp's sign message")
	}
	if !types.VerifyP2pSignBytes(spP2pPubkey, nodeSign, signmsg) {
		return errors.New("failed verifying sp's signature")
	}
	return nil
}

// RspFileStorageInfoVerifier task level verifier for all messages carrying RspFileStorageInfo in a download task
func RspFileStorageInfoVerifier(ctx context.Context, cmd string, target interface{}) error {
	err := verifyReqId(ctx, cmd)
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
		err := verifySpP2pAddress(ctx, cmd)
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
func RspFileStorageInfoWithNoReqIdVerifier(ctx context.Context, cmd string, target interface{}) error {
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

func verifyReqId(ctx context.Context, cmd string) error {
	reqCmd, found := p2pserver.GetP2pServer(ctx).LoadReqId(requests.GetReqIdFromMessage(ctx))
	if !found {
		return errors.New("no previous request for this rsp found")
	}
	if strings.Compare(strings.Replace(reqCmd, "Req", "Rsp", 1), cmd) != 0 {
		return errors.New("message types don't match")
	}
	return nil
}

func PpRspVerifier(ctx context.Context, cmd string, target interface{}) error {
	return verifyReqId(ctx, cmd)
}

func SpRspVerifier(ctx context.Context, cmd string, target interface{}) error {
	if err := verifySpP2pAddress(ctx, cmd); err != nil {
		return err
	}
	return verifyReqId(ctx, cmd)
}

func SpAddressVerifier(ctx context.Context, cmd string, target interface{}) error {
	return verifySpP2pAddress(ctx, cmd)
}
