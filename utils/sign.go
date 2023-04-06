package utils

import (
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
)

// GetFileUploadWalletSignMessage upload: wallet sign message for file upload request from the (rpc or cmd) user
func GetFileUploadWalletSignMessage(fileHash, walletAddr, sn string) string {
	return fileHash + walletAddr + sn
}

// GetFileDownloadWalletSignMessage download: wallet sign message for download request from the (rpc or cmd) user
func GetFileDownloadWalletSignMessage(fileHash, walletAddr, sn string) string {
	return fileHash + walletAddr + sn
}

// GetRspUploadFileSpNodeSignMessage upload: node sign message for upload file response, between sp and uploader pp, the dest pp verify this too
func GetRspUploadFileSpNodeSignMessage(rspMsg *protos.RspUploadFile) ([]byte, error) {
	msg, err := proto.Marshal(rspMsg)
	if err != nil {
		return nil, err
	}
	return CalcHashBytes(msg), nil
}

// GetRspBackupFileSpNodeSignMessage
func GetRspBackupFileSpNodeSignMessage(rspMsg *protos.RspBackupStatus) ([]byte, error) {
	msg, err := proto.Marshal(rspMsg)
	if err != nil {
		return nil, err
	}
	return CalcHashBytes(msg), nil
}

// GetReqBackupSliceNoticeSpNodeSignMessage
func GetReqBackupSliceNoticeSpNodeSignMessage(reqMsg *protos.ReqFileSliceBackupNotice) ([]byte, error) {
	msg, err := proto.Marshal(reqMsg)
	if err != nil {
		return nil, err
	}
	return CalcHashBytes(msg), nil
}

// GetRspFileStorageInfoNodeSignMessage download: node sign message for download file response, between sp and downloader pp
func GetRspFileStorageInfoNodeSignMessage(rspMsg *protos.RspFileStorageInfo) ([]byte, error) {
	msg, err := proto.Marshal(rspMsg)
	if err != nil {
		return nil, err
	}
	return CalcHashBytes(msg), nil
}

// GetFileReplicaInfoWalletSignMessage replica info: wallet sign message for get file replica info request from the (rpc or cmd) user
func GetFileReplicaInfoWalletSignMessage(fileHash, walletAddr string) string {
	return fileHash + walletAddr
}
