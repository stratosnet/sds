package utils

import (
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/msg/protos"
)

// GetFileUploadWalletSignMessage upload: wallet sign message for file upload request from the (rpc or cmd) user
func GetFileUploadWalletSignMessage(fileHash, walletAddr, sn string) string {
	return fileHash + walletAddr + sn
}

// GetReqUploadFileNodeSignMessage upload: node sign message for upload file request, between uploader pp and sp
func GetReqUploadFileNodeSignMessage(p2pAddr, fileHash, msgTypeStr string) string {
	return p2pAddr + fileHash + msgTypeStr
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

// GetReqUploadFileSlicePpNodeSignMessage upload: node sign message for upload slice request, between uploader pp and storage pp
func GetReqUploadFileSlicePpNodeSignMessage(srcPpP2pAddr, destPpP2pAddr, msgTypeStr, timeStamp string) string {
	return srcPpP2pAddr + destPpP2pAddr + msgTypeStr + timeStamp
}

// GetReportUploadSliceResultPpNodeSignMessage upload: node sign message for report upload slice result, between storage pp and sp, or uploader pp and sp
func GetReportUploadSliceResultPpNodeSignMessage(p2pAddr, fileHash, sliceHash, taskId, msgTypeStr string) string {
	return p2pAddr + fileHash + sliceHash + taskId + msgTypeStr
}

// GetRspUploadFileSliceNodeSignMessage upload: node sign message for upload slice response, between storage pp and uploader pp
func GetRspUploadFileSliceNodeSignMessage(srcP2pAddr, destP2pAddr, msgTypeStr string) string {
	return srcP2pAddr + destP2pAddr + msgTypeStr
}

// GetFileDownloadWalletSignMessage download: wallet sign message for download request from the (rpc or cmd) user
func GetFileDownloadWalletSignMessage(fileHash, walletAddr, sn string) string {
	return fileHash + walletAddr + sn
}

// GetReqFileStorageInfoNodeSignMessage download: node sign message for download file request, between downloader pp and sp
func GetReqFileStorageInfoNodeSignMessage(ppP2pAddr, filePath, msgTypeStr string) string {
	return ppP2pAddr + filePath + msgTypeStr
}

// GetRspFileStorageInfoNodeSignMessage download: node sign message for download file response, between sp and downloader pp
func GetRspFileStorageInfoNodeSignMessage(rspMsg *protos.RspFileStorageInfo) ([]byte, error) {
	msg, err := proto.Marshal(rspMsg)
	if err != nil {
		return nil, err
	}
	return CalcHashBytes(msg), nil
}

// GetReqDownloadSlicePpNodeSignMessage download: node sign message for download slice request, between downloader pp to storage pp
func GetReqDownloadSlicePpNodeSignMessage(srcPpP2pAddr, destPpP2pAddr, sliceHash, msgTypeStr, timeStamp string) string {
	return srcPpP2pAddr + destPpP2pAddr + sliceHash + msgTypeStr + timeStamp
}

// GetReportDownloadSliceResultPpNodeSignMessage download: node sign message for download slice request, between downloader pp to storage pp
func GetReportDownloadSliceResultPpNodeSignMessage(p2pAddress, sliceHash, taskId, msgTypeStr string) string {
	return p2pAddress + sliceHash + taskId + msgTypeStr
}

// GetFileDownloadShareNodeSignMessage share: node sign message for download shared file request, between pp to sp
func GetFileDownloadShareNodeSignMessage(p2pAddr, shareLink, msgTypeStr string) string {
	return p2pAddr + shareLink + msgTypeStr
}

// GetReqTransferDownloadPpNodeSignMessage transfer: node sign message for transfer download slice request, between downloader pp to storage pp
func GetReqTransferDownloadPpNodeSignMessage(srcPpP2pAddr, destPpP2pAddr, sliceHash, msgTypeStr string) string {
	return srcPpP2pAddr + destPpP2pAddr + sliceHash + msgTypeStr
}

// GetReqTransferDownloadPpNodeSignMessage download: node sign message for download slice request, between downloader pp to storage pp
func GetRspTransferDownloadPpNodeSignMessage(srcPpP2pAddr, spP2pAddr, sliceHash, msgTypeStr string) string {
	return srcPpP2pAddr + spP2pAddr + sliceHash + msgTypeStr
}

// GetReqReportBackupSliceResultNodeSignMessage download: node sign message for download slice request, between downloader pp to storage pp
func GetReqReportBackupSliceResultNodeSignMessage(srcPpP2pAddr, spP2pAddr, sliceHash, msgTypeStr string) string {
	return srcPpP2pAddr + spP2pAddr + sliceHash + msgTypeStr
}
