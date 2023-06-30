package utils

import (
	"github.com/stratosnet/sds/msg/protos"
	"google.golang.org/protobuf/proto"
)

// GetFileUploadWalletSignMessage upload: wallet sign message for file upload request from the (rpc or cmd) user
func GetFileUploadWalletSignMessage(fileHash, walletAddr, sn string, timestamp int64) string {
	return fileHash + walletAddr + sn + string(Int64ToByte(timestamp))
}

// GetFileDownloadWalletSignMessage download: wallet sign message for download request from the (rpc or cmd) user
func GetFileDownloadWalletSignMessage(fileHash, walletAddr, sn string, timestamp int64) string {
	return fileHash + walletAddr + sn + string(Int64ToByte(timestamp))
}

// GetRspUploadFileSpNodeSignMessage upload: node sign message for upload file response, between sp and uploader pp, the dest pp verify this too
func GetRspUploadFileSpNodeSignMessage(rspMsg *protos.RspUploadFile) ([]byte, error) {
	msg, err := proto.Marshal(rspMsg)
	if err != nil {
		return nil, err
	}
	return CalcHashBytes(msg), nil
}

func GetRspBackupFileSpNodeSignMessage(rspMsg *protos.RspBackupStatus) ([]byte, error) {
	msg, err := proto.Marshal(rspMsg)
	if err != nil {
		return nil, err
	}
	return CalcHashBytes(msg), nil
}

func GetNoticeFileSliceBackupSpNodeSignMessage(reqMsg *protos.NoticeFileSliceBackup) ([]byte, error) {
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
func GetFileReplicaInfoWalletSignMessage(fileHash, walletAddr string, timestamp int64) string {
	return fileHash + walletAddr + string(Int64ToByte(timestamp))
}

func DeleteFileWalletSignMessage(fileHash, walletAddr string, timestamp int64) string {
	return fileHash + walletAddr + string(Int64ToByte(timestamp))
}
func DeleteShareWalletSignMessage(shareId, walletAddr string, timestamp int64) string {
	return shareId + walletAddr + string(Int64ToByte(timestamp))
}
func FindMyFileListWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + string(Int64ToByte(timestamp))
}
func GetShareFileWalletSignMessage(shareId, walletAddr string, timestamp int64) string {
	return shareId + walletAddr + string(Int64ToByte(timestamp))
}
func GetSPListWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + string(Int64ToByte(timestamp))
}
func PrepayWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + string(Int64ToByte(timestamp))
}
func RegisterWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + string(Int64ToByte(timestamp))
}
func RegisterNewPPWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + string(Int64ToByte(timestamp))
}
func ShareFileWalletSignMessage(fileHash, walletAddr string, timestamp int64) string {
	return fileHash + walletAddr + string(Int64ToByte(timestamp))
}
func ShareLinkWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + string(Int64ToByte(timestamp))
}
func ClearExpiredShareLinksWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + string(Int64ToByte(timestamp))
}
