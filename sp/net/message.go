package net

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	"github.com/stratosnet/sds/relay/sds"
	"github.com/stratosnet/sds/relay/stratoschain"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stratosnet/sds/msg/header"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/sp/common"
	"github.com/stratosnet/sds/sp/storages/table"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/hashring"
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
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			msgInStr, _ := m.server.GetCache().DeQueue("msg_queue")
			if msgInStr != nil && msgInStr != "" {
				switch msgInStr.(type) {
				case string:
					msg := new(common.MsgWrapper)
					if json.Unmarshal([]byte(msgInStr.(string)), msg) == nil {
						m.msgQueue <- *&msg.Msg
					}
				}
			}
		}
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
				msgMining := msg.(*common.MsgMining)
				m.Mining(msgMining.P2PAddress, msgMining.NetworkAddress, msgMining.Name, msgMining.Puk)
			} else if msgType == common.MSG_TRANSFER_NOTICE {
				msgTransferNotice := msg.(*common.MsgTransferNotice)
				m.TransferNotice(msgTransferNotice.SliceHash, msgTransferNotice.FromP2PAddress,
					msgTransferNotice.ToP2PAddress, msgTransferNotice.DeleteOrigin)
			} else if msgType == common.MSG_BACKUP_SLICE {
				msgBackupSlice := msg.(*common.MsgBackupSlice)
				m.BackupSlice(msgBackupSlice.SliceHash, msgBackupSlice.FromP2PAddress)
			} else if msgType == common.MSG_BACKUP_PP {
				m.BackupPP(msg.(*common.MsgBackupPP).P2PAddress)
			} else if msgType == common.MSG_DELETE_SLICE {
				msgDeleteSlice := msg.(*common.MsgDeleteSlice)
				m.DeleteSlice(msgDeleteSlice.P2PAddress, msgDeleteSlice.SliceHash)
			} else if msgType == common.MSG_AGGREGATE_TRAFFIC {
				msgAggregateTraffic := msg.(*common.MsgAggregateTraffic)
				aggregatedTraffic, err := m.AggregateTraffic(msgAggregateTraffic.Time)
				if err != nil {
					utils.ErrorLog("Error when aggregating Traffic: ", err)
				}
				utils.Log(aggregatedTraffic)
			}
		}
	}

}

// Mining
func (m *MsgHandler) Mining(p2pAddress, networkAddress, name string, puk []byte) {
	if !m.server.HashRing.IsOnline(p2pAddress) {
		node := &hashring.Node{
			ID:   p2pAddress,
			Host: networkAddress,
		}
		m.server.HashRing.AddNode(node)
	}

	m.server.HashRing.SetOnline(p2pAddress)

	user := &table.User{P2pAddress: p2pAddress}
	if m.server.CT.Fetch(user) == nil {
		user.Name = name
		m.server.CT.Save(user)
	}

	pp := &table.PP{P2pAddress: p2pAddress}
	if m.server.CT.Fetch(pp) == nil {
		pp.State = table.STATE_ONLINE
		pp.NetworkAddress = networkAddress
		pp.PubKey = fmt.Sprintf("PubKeySecp256k1{%X}", puk)
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

	p2pAddress := m.server.Who(name)
	if p2pAddress == "" {
		return
	}
	m.server.HashRing.SetOffline(p2pAddress)

	pp := &table.PP{P2pAddress: p2pAddress}
	if m.server.CT.Fetch(pp) == nil {
		pp.State = table.STATE_OFFLINE
		m.server.CT.Save(pp)
	}

	m.server.RmConn(name)

	utils.Log(fmt.Sprintf("!!! %s@%s disconnect, current online nodes: %d", p2pAddress, name, m.server.HashRing.NodeOkCount))

}

// BackupPP
func (m *MsgHandler) BackupPP(p2pAddress string) {

	if p2pAddress == "" {
		utils.Log("BackupPP: msg data given incorrect ")
		utils.Log("p2pAddress = ", p2pAddress)
		return
	}

	res, err := m.server.CT.FetchTables([]table.FileSlice{}, map[string]interface{}{
		"alias": "fs",
		"join":  []string{"file_slice_storage", "fs.slice_hash = fss.slice_hash", "fss"},
		"where": map[string]interface{}{
			"fss.p2p_address = ?": p2pAddress,
		},
	})

	if err != nil {
		return
	}
	fileSlices := res.([]table.FileSlice)

	for index, fs := range fileSlices {

		key := fs.SliceHash + "#" + strconv.FormatUint(fs.SliceNumber, 10) + "#" + strconv.Itoa(index)

		_, newStoreP2PAddress := m.server.HashRing.GetNodeExcludedNodeIDs(key, m.server.System.MissingBackupWalletAddr)

		if newStoreP2PAddress != "" && fs.P2pAddress != newStoreP2PAddress {
			m.TransferNotice(fs.SliceHash, fs.P2pAddress, newStoreP2PAddress, false)
		}
	}

}

// TransferNotice
func (m *MsgHandler) TransferNotice(sliceHash, fromP2PAddress, toP2PAddress string, deleteOrigin bool) {
	if sliceHash == "" || fromP2PAddress == "" || toP2PAddress == "" {
		utils.Log("TransferNotice: msg data given incorrect")
		utils.Log("sliceHash:", sliceHash)
		utils.Log("fromP2PAddress", fromP2PAddress)
		utils.Log("toP2PAddress", toP2PAddress)
		utils.Log("DeleteOrigin:", deleteOrigin)
		return
	}

	fromUser := &table.User{P2pAddress: fromP2PAddress}
	if m.server.CT.Fetch(fromUser) != nil {
		utils.Log(fmt.Sprintf("Couldn't find origin P2P address %v in database", fromP2PAddress))
		return
	}

	toUser := &table.User{P2pAddress: toP2PAddress}
	if m.server.CT.Fetch(toUser) != nil {
		utils.Log(fmt.Sprintf("Couldn't find destination P2P address %v in database", toP2PAddress))
		return
	}

	fileSlice := &table.FileSlice{
		FileSliceStorage: table.FileSliceStorage{
			P2pAddress: fromP2PAddress,
		},
		SliceHash: sliceHash,
	}
	if m.server.CT.Fetch(fileSlice) != nil {
		utils.Log(errors.New("no slice found"))
		return
	}

	// get online PP info todo change to read from redis
	node := m.server.HashRing.Node(toP2PAddress)
	if node == nil || node.Host == "" {
		utils.Log("TransferNotice: new PP[", toP2PAddress, "] is not online")
		return
	}
	transferCer := utils.CalcHash([]byte(fileSlice.SliceHash + "#" + toP2PAddress + "#" + strconv.FormatInt(time.Now().UnixNano(), 10)))

	req := &protos.ReqTransferNotice{
		TransferCer: transferCer,
		FromSp:      true,
		SliceStorageInfo: &protos.SliceStorageInfo{
			SliceSize: fileSlice.SliceSize,
			SliceHash: fileSlice.SliceHash,
		},
		StoragePpInfo: &protos.PPBaseInfo{
			P2PAddress:     fromUser.P2pAddress,
			WalletAddress:  fromUser.WalletAddress,
			NetworkAddress: fromUser.NetworkAddress,
		},
		DeleteOrigin: deleteOrigin,
	}

	transferRecord := &table.TransferRecord{
		SliceHash:          fileSlice.SliceHash,
		SliceSize:          fileSlice.SliceSize,
		TransferCer:        transferCer,
		FromP2pAddress:     fromUser.P2pAddress,
		FromWalletAddress:  fromUser.WalletAddress,
		FromNetworkAddress: fromUser.NetworkAddress,
		ToP2pAddress:       toUser.P2pAddress,
		ToWalletAddress:    toUser.WalletAddress,
		ToNetworkAddress:   toUser.NetworkAddress,
		Status:             table.TRANSFER_RECORD_STATUS_CHECK,
		Time:               0,
	}

	// todo change to read from redis
	m.server.Store(transferRecord, 3600*time.Second)

	m.server.SendMsg(node.ID, header.ReqTransferNotice, req)
}

type AggregatedTraffic struct {
	P2PAddress string
	Volume     uint64
}

func (m *MsgHandler) AggregateTraffic(time int64) ([]AggregatedTraffic, error) {
	type TrafficGroup struct {
		table.Traffic
		P2PAddress  string
		TotalVolume string
	}

	res, err := m.server.CT.FetchTables([]TrafficGroup{}, map[string]interface{}{
		"columns": "provider_p2p_address AS p2p_address, SUM(volume) AS total_volume",
		"where": map[string]interface{}{
			"delivery_time < ?": time,
		},
		"groupBy": "provider_p2p_address",
		"orderBy": "total_volume desc",
	})

	if err != nil {
		return []AggregatedTraffic{}, err
	}

	trafficGroups := res.([]TrafficGroup)
	aggregatedTraffic := make([]AggregatedTraffic, 0, len(trafficGroups))
	rewardAccounts := make([]interface{}, 0, len(trafficGroups))

	if len(trafficGroups) > 0 {
		totalVolume := uint64(0)
		aggregatedVolume := uint64(0)

		for _, group := range trafficGroups {
			volume, _ := strconv.Atoi(group.TotalVolume)
			totalVolume += uint64(volume)
		}
		threshold := uint64(float64(totalVolume) * 0.8)

		for _, group := range trafficGroups {
			if aggregatedVolume <= threshold {
				volume, _ := strconv.Atoi(group.TotalVolume)
				aggregatedVolume += uint64(volume)
				aggregatedTraffic = append(aggregatedTraffic, AggregatedTraffic{
					P2PAddress: group.P2PAddress,
					Volume:     uint64(volume),
				})
				rewardAccounts = append(rewardAccounts, group.P2PAddress)
			} else {
				break
			}
		}

		res, err := m.server.CT.FetchTables([]table.Traffic{}, map[string]interface{}{
			"where": map[string]interface{}{
				"provider_p2p_address in (?" + strings.Repeat(",?", len(rewardAccounts)-1) + ")": rewardAccounts,
				"delivery_time < ?": time,
			},
		})
		trafficRecords := res.([]table.Traffic)
		if err != nil {
			return []AggregatedTraffic{}, err
		}

		//TODO persist the file to the SDS
		fileName := fmt.Sprintf("tmp/traffic_aggregation_%v.csv", time)
		err = m.WriteTrafficToCsvFile(fileName, trafficRecords)
		if err != nil {
			return []AggregatedTraffic{}, err
		}

		// Calculate epoch
		var epoch uint64 = 0
		epochVar := &table.Variable{Name: "epoch"}
		if m.server.CT.Fetch(epochVar) == nil {
			epoch, err = strconv.ParseUint(epochVar.Value, 10, 64)
		}
		epoch++
		epochVar.Value = strconv.FormatUint(epoch, 10)
		if err := m.server.CT.Save(epochVar); err != nil {
			utils.ErrorLog("Couldn't save aggregateTraffic epoch to database: " + err.Error())
		}

		// Send volume report transaction
		fileHash := utils.CalcFileHash(fileName)
		err = broadcastVolumeReportTx(trafficRecords, epoch, fileHash, m.server)
		if err != nil {
			return []AggregatedTraffic{}, err
		}

		num := m.server.CT.GetDriver().Delete("traffic", map[string]interface{}{
			"provider_p2p_address in (?" + strings.Repeat(",?", len(rewardAccounts)-1) + ")": rewardAccounts,
			"delivery_time < ?": time,
		})

		if num == 0 {
			return []AggregatedTraffic{}, errors.New("cannot delete traffic records for rewarded accounts")
		}
	}

	return aggregatedTraffic, nil
}

func (m *MsgHandler) WriteTrafficToCsvFile(fileName string, records []table.Traffic) error {
	file, err := os.Create(fileName)
	if err != nil {
		return errors.New(fmt.Sprint("cannot create file ", err.Error()))
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for idx, value := range records {
		if idx == 0 {
			err := writer.Write(value.GetHeaders())
			if err != nil {
				return errors.New(fmt.Sprint("cannot write to file", err.Error()))
			}
		}

		err := writer.Write(value.ToSlice())
		if err != nil {
			return errors.New(fmt.Sprint("cannot write to file", err.Error()))
		}
	}

	return nil
}

// DeleteSlice from P or PP
func (m *MsgHandler) DeleteSlice(p2pAddress, sliceHash string) {

	if sliceHash == "" || p2pAddress == "" {
		utils.Log("DeleteSlice: msg data given incorrect ")
		utils.Log("sliceHash = ", sliceHash)
		utils.Log("P2pAddress = ", p2pAddress)
		return
	}

	req := &protos.ReqDeleteSlice{
		P2PAddress: p2pAddress,
		SliceHash:  sliceHash,
	}

	m.server.SendMsg(p2pAddress, header.ReqTransferNotice, req)
}

// BackupSlice
func (m *MsgHandler) BackupSlice(sliceHash, fileSliceP2PAddress string) {

	if sliceHash == "" || fileSliceP2PAddress == "" {
		utils.Log("BackupSlice: msg data given incorrect ")
		utils.Log("sliceHash = ", sliceHash)
		utils.Log("fileSliceP2PAddress = ", fileSliceP2PAddress)
		return
	}

	up, down := m.server.HashRing.GetNodeUpDownNodes(fileSliceP2PAddress)
	utils.DebugLog("Up stream ", up)
	utils.DebugLog("Down stream ", down)

	// if both up and down stream is empty, only have 1 node
	if up != "" && down != "" && fileSliceP2PAddress != up && fileSliceP2PAddress != down {
		// backup to up stream first
		m.TransferNotice(sliceHash, fileSliceP2PAddress, up, false)
		if up != down {
			// if up and down are not the same, backup to down stream
			m.TransferNotice(sliceHash, fileSliceP2PAddress, down, false)
		}
	}

}

// NewMsgHandler
func NewMsgHandler(server *Server) *MsgHandler {
	return &MsgHandler{
		msgQueue: make(chan common.Msg, 10),
		server:   server,
	}
}

func broadcastVolumeReportTx(traffic []table.Traffic, epoch uint64, fileHash string, s *Server) error {
	spPubKey := s.WalletPrivateKey.PubKey()
	spPrivKey := s.WalletPrivateKey.(secp256k1.PrivKeySecp256k1)
	spWalletAddress := spPubKey.Address()
	spWalletAddressString := types.AccAddress(spPubKey.Address()).String()

	txMsg, err := stratoschain.BuildVolumeReportMsg(traffic, spWalletAddress, epoch, fileHash)
	if err != nil {
		return err
	}

	txBytes, err := stratoschain.BuildTxBytes(s.Conf.BlockchainInfo.Token, s.Conf.BlockchainInfo.ChainId, "",
		spWalletAddressString, "sync", txMsg, s.Conf.BlockchainInfo.Transactions.Fee,
		s.Conf.BlockchainInfo.Transactions.Gas, spPrivKey[:])
	if err != nil {
		return err
	}

	relayMsg := &protos.RelayMessage{
		Type: sds.TypeBroadcast,
		Data: txBytes,
	}
	msgBytes, err := proto.Marshal(relayMsg)
	if err != nil {
		return err
	}

	s.SubscriptionServer.Broadcast("broadcast", msgBytes)
	return nil
}
