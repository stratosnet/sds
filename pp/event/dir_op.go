package event

import (
	"context"
	"fmt"
	"github.com/qsnetwork/qsds/framework/spbf"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/pp/client"
	"github.com/qsnetwork/qsds/pp/setting"
	"github.com/qsnetwork/qsds/utils"
	"net/http"
	"strings"
)

// NowDir current dir
var NowDir = ""

// ReqMakeDirectory ReqMakeDirectory
func ReqMakeDirectory(ctx context.Context, conn spbf.WriteCloser) {
	// pp send to SP
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspMakeDirectory RspMakeDirectory
func RspMakeDirectory(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.RspMakeDirectory
	if unmarshalData(ctx, &target) {
		if target.WalletAddress == setting.WalletAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPMkdir, &target)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}
	}
}

// MakeDirectory
func MakeDirectory(path, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqMakeDirectoryData(path, reqID), header.ReqMakeDirectory)
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// RemoveDirectory
func RemoveDirectory(path, reqID string, w http.ResponseWriter) {
	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqRemoveDirectoryData(path, reqID), header.ReqRemoveDirectory)
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqRemoveDirectory ReqRemoveDirectory
func ReqRemoveDirectory(ctx context.Context, conn spbf.WriteCloser) {
	// pp send to SP
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspRemoveDirectory RspRemoveDirectory
func RspRemoveDirectory(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.RspRemoveDirectory
	if unmarshalData(ctx, &target) {
		if target.WalletAddress == setting.WalletAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPRMdir, &target)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}
	}
}

// MoveFileDirectory
func MoveFileDirectory(fileHash, originalDir, targetDir, reqID string, w http.ResponseWriter) {
	utils.DebugLog("MoveFileDirectory fileHash", fileHash, "originalDir", originalDir, "targetDir", targetDir, reqID)

	if setting.CheckLogin() {
		sendMessage(client.PPConn, reqMoveFileDirectoryData(fileHash, originalDir, targetDir, reqID), header.ReqMoveFileDirectory)
		stroeResponseWriter(reqID, w)
	} else {
		notLogin(w)
	}
}

// ReqMoveFileDirectory ReqMoveFileDirectory
func ReqMoveFileDirectory(ctx context.Context, conn spbf.WriteCloser) {
	utils.DebugLog("ReqMoveFileDirectory")
	transferSendMessageToSPServer(spbf.MessageFromContext(ctx))
}

// RspMoveFileDirectory RspMoveFileDirectory
func RspMoveFileDirectory(ctx context.Context, conn spbf.WriteCloser) {
	var target protos.RspMoveFileDirectory
	if unmarshalData(ctx, &target) {
		if target.WalletAddress == setting.WalletAddress {
			if target.Result.State == protos.ResultState_RES_SUCCESS {
				fmt.Println("action  successfully", target.Result.Msg)
			} else {
				fmt.Println("action  failed", target.Result.Msg)
			}
			putData(target.ReqId, HTTPMVdir, &target)
		} else {
			transferSendMessageToClient(target.WalletAddress, spbf.MessageFromContext(ctx))
		}
	}
}

// Goto cd
func Goto(dir string) {
	if dir == "~" {
		NowDir = ""
	} else if dir == ".." {
		// go back to upper level
		strs := strings.Split(NowDir, "/")
		utils.DebugLog("strsstrs = ", strs)
		if len(strs) < 2 {
			NowDir = "" // root directory
		} else {
			newDir := ""
			for index := 0; index < len(strs); index++ {
				if index == 0 {
					newDir = strs[0]
				} else if index != len(strs)-1 {
					newDir += ("/" + strs[0])
				}
			}
			NowDir = newDir
		}
	} else {
		if NowDir != "" {
			NowDir += ("/" + dir)
		} else {
			NowDir += dir
		}

	}

	if NowDir == "" {
		fmt.Println("current dir：root")
	} else {
		fmt.Println("current dir：", NowDir)
	}

}
