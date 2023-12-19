package utils

import (
	"strconv"
)

// GetFileUploadWalletSignMessage upload: wallet sign message for file upload request from the (rpc or cmd) user
func GetFileUploadWalletSignMessage(fileHash, walletAddr, sn string, timestamp int64) string {
	return fileHash + walletAddr + sn + strconv.FormatInt(timestamp, 10)
}

// GetFileDownloadWalletSignMessage download: wallet sign message for download request from the (rpc or cmd) user
func GetFileDownloadWalletSignMessage(fileHash, walletAddr, sn string, timestamp int64) string {
	return fileHash + walletAddr + sn + strconv.FormatInt(timestamp, 10)
}

// GetFileReplicaInfoWalletSignMessage replica info: wallet sign message for get file replica info request from the (rpc or cmd) user
func GetFileReplicaInfoWalletSignMessage(fileHash, walletAddr string, timestamp int64) string {
	return fileHash + walletAddr + strconv.FormatInt(timestamp, 10)
}

func GetFileStatusWalletSignMessage(fileHash, walletAddr string, timestamp int64) string {
	return fileHash + walletAddr + strconv.FormatInt(timestamp, 10)
}

func DeleteFileWalletSignMessage(fileHash, walletAddr string, timestamp int64) string {
	return fileHash + walletAddr + strconv.FormatInt(timestamp, 10)
}
func DeleteShareWalletSignMessage(shareId, walletAddr string, timestamp int64) string {
	return shareId + walletAddr + strconv.FormatInt(timestamp, 10)
}
func FindMyFileListWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + strconv.FormatInt(timestamp, 10)
}
func GetShareFileWalletSignMessage(shareId, walletAddr string, timestamp int64) string {
	return shareId + walletAddr + strconv.FormatInt(timestamp, 10)
}
func GetSPListWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + strconv.FormatInt(timestamp, 10)
}
func PrepayWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + strconv.FormatInt(timestamp, 10)
}
func RegisterWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + strconv.FormatInt(timestamp, 10)
}
func RegisterNewPPWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + strconv.FormatInt(timestamp, 10)
}
func ShareFileWalletSignMessage(fileHash, walletAddr string, timestamp int64) string {
	return fileHash + walletAddr + strconv.FormatInt(timestamp, 10)
}
func ShareLinkWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + strconv.FormatInt(timestamp, 10)
}
func ClearExpiredShareLinksWalletSignMessage(walletAddr string, timestamp int64) string {
	return walletAddr + strconv.FormatInt(timestamp, 10)
}
