package utils

// GetFileUploadWalletSignMessage upload: wallet sign message for file upload request from the (rpc or cmd) user
func GetFileUploadWalletSignMessage(fileHash, walletAddr string) string {
	return fileHash + walletAddr
}

// GetReqUploadFileNodeSignMessage upload: node sign message for upload file request, between uploader pp and sp
func GetReqUploadFileNodeSignMessage(p2pAddr, fileHash, msgTypeStr string) string {
	return p2pAddr + fileHash + msgTypeStr
}

// GetRspUploadFileSpNodeSignMessage upload: node sign message for upload file response, between sp and uploader pp
func GetRspUploadFileSpNodeSignMessage(ppP2pAddr, spP2PAddress, fileHash, msgTypeStr string) string {
	return ppP2pAddr + spP2PAddress + fileHash + msgTypeStr
}

// GetReqUploadFileSliceSpNodeSignMessage upload: node sign message for upload slice request, between sp and storage pp through uploader pp
func GetReqUploadFileSliceSpNodeSignMessage(ppP2pAddr, spP2PAddress, fileHash, msgTypeStr string) string {
	return ppP2pAddr + spP2PAddress + fileHash + msgTypeStr
}

// GetReqUploadFileSlicePpNodeSignMessage upload: node sign message for upload slice request, between uploader pp and storage pp
func GetReqUploadFileSlicePpNodeSignMessage(srcPpP2pAddr, destPpP2pAddr, msgTypeStr string) string {
	return srcPpP2pAddr + destPpP2pAddr + msgTypeStr
}

// GetReportUploadSliceResultPpNodeSignMessage upload: node sign message for report upload slice result, between storage pp and sp, or uploader pp and sp
func GetReportUploadSliceResultPpNodeSignMessage(p2pAddr, fileHash, msgTypeStr string) string {
	return p2pAddr + fileHash + msgTypeStr
}

// GetRspUploadFileSliceNodeSignMessage upload: node sign message for upload slice response, between storage pp and uploader pp
func GetRspUploadFileSliceNodeSignMessage(srcP2pAddr, destP2pAddr, msgTypeStr string) string {
	return srcP2pAddr + destP2pAddr + msgTypeStr
}

// GetFileDownloadWalletSignMessage download: wallet sign message for download request from the (rpc or cmd) user
func GetFileDownloadWalletSignMessage(fileHash, walletAddr string) string {
	return fileHash + walletAddr
}

// GetReqFileStorageInfoNodeSignMessage download: node sign message for download file request, between downloader pp and sp
func GetReqFileStorageInfoNodeSignMessage(ppP2pAddr, filePath, msgTypeStr string) string {
	return ppP2pAddr + filePath + msgTypeStr
}

// GetRspFileStorageInfoNodeSignMessage download: node sign message for download file response, between sp and downloader pp
func GetRspFileStorageInfoNodeSignMessage(ppP2pAddr, spP2PAddress, fileHash, msgTypeStr string) string {
	return ppP2pAddr + spP2PAddress + fileHash + msgTypeStr
}

// GetReqDownloadSliceSpNodeSignMessage download: node sign message for download slice request, between sp and storage pp through downloader pp
func GetReqDownloadSliceSpNodeSignMessage(ppP2pAddr, spP2pAddr, sliceHash, msgTypeStr string) string {
	return ppP2pAddr + spP2pAddr + sliceHash + msgTypeStr
}

// GetReqDownloadSlicePpNodeSignMessage download: node sign message for download slice request, between downloader pp to storage pp
func GetReqDownloadSlicePpNodeSignMessage(srcPpP2pAddr, destPpP2pAddr, sliceHash, msgTypeStr string) string {
	return srcPpP2pAddr + destPpP2pAddr + sliceHash + msgTypeStr
}

// GetReportDownloadSliceResultPpNodeSignMessage download: node sign message for download slice request, between downloader pp to storage pp
func GetReportDownloadSliceResultPpNodeSignMessage(p2pAdress, sliceHash, msgTypeStr string) string {
	return p2pAdress + sliceHash + msgTypeStr
}

// GetFileDownloadShareWalletSignMessage share: wallet sign message for download shared file request from the (rpc or cmd) user
// this message must be the same as GetFileDownloadWalletSignMessage()
func GetFileDownloadShareWalletSignMessage(fileHash, walletAddr string) string {
	return fileHash + walletAddr
}

// GetFileDownloadShareNodeSignMessage share: node sign message for download shared file request, between pp to sp
func GetFileDownloadShareNodeSignMessage(p2pAddr, shareLink, msgTypeStr string) string {
	return p2pAddr + shareLink + msgTypeStr
}
