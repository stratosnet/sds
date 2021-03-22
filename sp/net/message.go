package net

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qsnetwork/qsds/msg/header"
	"github.com/qsnetwork/qsds/msg/protos"
	"github.com/qsnetwork/qsds/sp/common"
	"github.com/qsnetwork/qsds/sp/storages/table"
	"github.com/qsnetwork/qsds/utils"
	"github.com/qsnetwork/qsds/utils/hashring"
	"strconv"
	"time"
)

// MsgHandler
type MsgHandler struct {
	server   *Server
	msgQueue chan common.Msg
}

// AddMsg
func (m *MsgHandler) AddMsg(msg common.Msg) {
	m.msgQueue <- msg
}

// ListenMsgQueue
func (m *MsgHandler) ListenMsgQueue() {

	for {

		msgInStr, _ := m.server.GetCache().DeQueue("msg_queue")
		if msgInStr != nil && msgInStr != "" {
			switch msgInStr.(type) {
			case string:
				msg := new(common.Msg)
				if json.Unmarshal([]byte(msgInStr.(string)), msg) == nil {
					m.msgQueue <- *msg
				}
			}
		}

		time.Sleep(time.Second * 1)
	}
}

// Run
func (m *MsgHandler) Run() {

	if m.server == nil {
		utils.ErrorLog("no object to service")
		return
	}

	go m.ListenMsgQueue()

	for {

		select {

		case msg := <-m.msgQueue:

			msgType := msg.GetType()

			if msgType == common.MSG_LOGOUT {
				m.Logout(msg.(*common.MsgLogout).Name)
			} else if msgType == common.MSG_MIMING {
				msgMing := msg.(*common.MsgMining)
				m.Mining(msgMing.WalletAddress, msgMing.NetworkAddress, msgMing.Name, msgMing.Puk)
			} else if msgType == common.MSG_TRANSFER_NOTICE {
				msgTransferNotice := msg.(*common.MsgTransferNotice)
				m.TransferNotice(msgTransferNotice.SliceHash, msgTransferNotice.FromWalletAddress, msgTransferNotice.ToWalletAddress)
			} else if msgType == common.MSG_BACKUP_SLICE {
				msgBackupSlice := msg.(*common.MsgBackupSlice)
				m.BackupSlice(msgBackupSlice.SliceHash, msgBackupSlice.FromWalletAddress)
			} else if msgType == common.MSG_BACKUP_PP {
				m.BackupPP(msg.(*common.MsgBackupPP).WalletAddress)
			} else if msgType == common.MSG_DELETE_SLICE {
				msgDeleteSlice := msg.(*common.MsgDeleteSlice)
				m.DeleteSlice(msgDeleteSlice.WalletAddress, msgDeleteSlice.SliceHash)
			}
		}
	}

}

// Mining
func (m *MsgHandler) Mining(walletAddress, networkAddress, name string, puk []byte) {


	if !m.server.HashRing.IsOnline(walletAddress) {
		node := &hashring.Node{
			ID:   walletAddress,
			Host: networkAddress,
		}
		m.server.HashRing.AddNode(node)
	}

	m.server.HashRing.SetOnline(walletAddress)


	user := &table.User{WalletAddress: walletAddress}
	if m.server.CT.Fetch(user) == nil {
		user.Name = name
		m.server.CT.Save(user)
	}

	//
	pp := &table.PP{WalletAddress: walletAddress}
	if m.server.CT.Fetch(pp) == nil {
		pp.State = table.STATE_ONLINE
		pp.NetworkAddress = networkAddress
		pp.PubKey = hex.EncodeToString(puk)
		m.server.CT.Save(pp)
	}
}

// Logout
func (m *MsgHandler) Logout(name string) {

	if name == "" {
		utils.Log("Offline: msg data given incorrect ")
		utils.Log("name = ", name)
		return
	}

	walletAddress := m.server.Who(name)
	if walletAddress != "" {


		m.server.HashRing.SetOffline(walletAddress)


		pp := &table.PP{WalletAddress: walletAddress}
		if m.server.CT.Fetch(pp) == nil {
			pp.State = table.STATE_OFFLINE
			m.server.CT.Save(pp)
		}


		m.server.RmConn(name)

		utils.Log(fmt.Sprintf("!!! %s@%s disconnect, current online nodes: %d", walletAddress, name, m.server.HashRing.NodeOkCount))
	}
}

// BackupPP
func (m *MsgHandler) BackupPP(walletAddress string) {

	if walletAddress == "" {
		utils.Log("BackupPP: msg data given incorrect ")
		utils.Log("walletAddress = ", walletAddress)
		return
	}

	res, err := m.server.CT.FetchTables([]table.FileSlice{}, map[string]interface{}{
		"where": map[string]interface{}{
			"wallet_address = ?": walletAddress,
		},
	})

	if err == nil {

		fileSlices := res.([]table.FileSlice)

		for index, fs := range fileSlices {

			key := fs.SliceHash + "#" + strconv.FormatUint(fs.SliceNumber, 10) + "#" + strconv.Itoa(index)

			_, newStorePPWalletAddress := m.server.HashRing.GetNodeMissNodeIDs(key, m.server.System.MissingBackupWalletAddr)

			if newStorePPWalletAddress != "" &&
				fs.WalletAddress != newStorePPWalletAddress {

				m.TransferNotice(fs.SliceHash, fs.WalletAddress, newStorePPWalletAddress)
			}
		}
	}
}

// TransferNotice
func (m *MsgHandler) TransferNotice(sliceHash, sliceInWalletAddress, newStorePPWalletAddress string) {

	if sliceHash == "" || sliceInWalletAddress == "" || newStorePPWalletAddress == "" {
		utils.Log("TransferNotice: msg data given incorrect")
		utils.Log("sliceHash:", sliceHash)
		utils.Log("sliceInWalletAddress", sliceInWalletAddress)
		utils.Log("newStorePPWalletAddress", newStorePPWalletAddress)
		return
	}

	fileSlice := new(table.FileSlice)
	fileSlice.WalletAddress = sliceInWalletAddress
	fileSlice.SliceHash = sliceHash
	if m.server.CT.Fetch(fileSlice) != nil {
		utils.Log(errors.New("no slice found"))
		return
	}

	// get online PP info todo change to read from redis
	node := m.server.HashRing.Node(newStorePPWalletAddress)
	if node != nil && node.Host != "" {

		transferCer := utils.CalcHash([]byte(fileSlice.SliceHash + "#" + newStorePPWalletAddress + "#" + strconv.FormatInt(time.Now().UnixNano(), 10)))

		req := &protos.ReqTransferNotice{
			TransferCer: transferCer,
			FromSp:      true,
			SliceStorageInfo: &protos.SliceStorageInfo{
				SliceSize: fileSlice.SliceSize,
				SliceHash: fileSlice.SliceHash,
			},
			StoragePpInfo: &protos.PPBaseInfo{
				WalletAddress:  fileSlice.WalletAddress,
				NetworkAddress: fileSlice.NetworkAddress,
			},
		}


		transferRecord := &table.TransferRecord{
			SliceHash:          fileSlice.SliceHash,
			TransferCer:        transferCer,
			FromWalletAddress:  fileSlice.WalletAddress,
			ToWalletAddress:    newStorePPWalletAddress,
			FromNetworkAddress: fileSlice.NetworkAddress,
			Status:             table.TRANSFER_RECORD_STATUS_CHECK,
			Time:               0,
		}

		// todo change to read from redis
		m.server.Store(transferRecord, 3600*time.Second)

		m.server.SendMsg(node.ID, header.ReqTransferNotice, req)
	} else {
		utils.Log("TransferNotice: new PP[", newStorePPWalletAddress, "] is not online")
	}
}

// DeleteSlice from P or PP
func (m *MsgHandler) DeleteSlice(walletAddress, sliceHash string) {

	if sliceHash == "" || walletAddress == "" {
		utils.Log("DeleteSlice: msg data given incorrect ")
		utils.Log("sliceHash = ", sliceHash)
		utils.Log("WalletAddress = ", walletAddress)
		return
	}

	req := &protos.ReqDeleteSlice{
		WalletAddress: walletAddress,
		SliceHash:     sliceHash,
	}

	m.server.SendMsg(walletAddress, header.ReqTransferNotice, req)
}

// BackupSlice
func (m *MsgHandler) BackupSlice(sliceHash, sliceInWalletAddress string) {

	if sliceHash == "" || sliceInWalletAddress == "" {
		utils.Log("BackupSlice: msg data given incorrect ")
		utils.Log("sliceHash = ", sliceHash)
		utils.Log("sliceInWalletAddress = ", sliceInWalletAddress)
		return
	}

	up, down := m.server.HashRing.GetNodeUpDownNodes(sliceInWalletAddress)

	// if both up and down stream is empty, only have 1 node
	if up != "" && down != "" && sliceInWalletAddress != up && sliceInWalletAddress != down {
		// backup to up stream first
		m.TransferNotice(sliceHash, sliceInWalletAddress, up)
		if up != down {
			// if up and down are not the same, backup to down stream
			m.TransferNotice(sliceHash, sliceInWalletAddress, down)
		}
	}

}

// NewMsgHandler
func NewMsgHandler(server *Server) *MsgHandler {

	mh := new(MsgHandler)
	mh.msgQueue = make(chan common.Msg, 10)
	mh.server = server

	return mh
}
