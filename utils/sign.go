package utils

// GetFileUploadWalletSignMessage wallet sign message from the (rpc or cmd) user
func GetFileUploadWalletSignMessage(fileHash, walletAddr string) string {
	return fileHash + walletAddr
}

// GetReqUploadFileNodeSignMessage node sign message in entry PP for ReqUploadFile
func GetReqUploadFileNodeSignMessage(ppWalletAddr, p2pAddr, fileHash, msgTypeStr string) string {
	return ppWalletAddr + p2pAddr + fileHash + msgTypeStr
}

// GetRspUploadFileSliceSpNodeSignMessage node sign message in SP for RspUploadFileSlice
func GetRspUploadFileSpNodeSignMessage(ppP2pAddr, spP2PAddress, fileHash, msgTypeStr string) string {
	return ppP2pAddr + spP2PAddress + fileHash + msgTypeStr
}

// GetReqUploadFileSliceSpNodeSignMessage node sign message in SP for each slice and will be used later
// in ReqUploadFileSlice
func GetReqUploadFileSliceSpNodeSignMessage(ppP2pAddr, spP2PAddress, fileHash, msgTypeStr string) string {
	return ppP2pAddr + spP2PAddress + fileHash + msgTypeStr
}

// GetReqUploadFileSlicePpNodeSignMessage
func GetReqUploadFileSlicePpNodeSignMessage(srcPpP2pAddr, destPpP2pAddr, msgTypeStr string) string {
	return srcPpP2pAddr + destPpP2pAddr + msgTypeStr
}

// GetReportUploadSliceResultPpNodeSignMessage
func GetReportUploadSliceResultPpNodeSignMessage(p2pAddr, fileHash, msgTypeStr string) string {
	return p2pAddr + fileHash + msgTypeStr
}

// GetRspUploadFileSliceNodeSignMessage
func GetRspUploadFileSliceNodeSignMessage(srcP2pAddr, destP2pAddr, msgTypeStr string) string {
	return srcP2pAddr + destP2pAddr + msgTypeStr
}
