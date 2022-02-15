package websocket

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/event"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

var mu = sync.Mutex{}

// SocketStart SocketStart
func SocketStart(conn net.Conn, upMap, downMap, m map[string]interface{}) error {
	type prog struct {
		TaskID   string  `json:"taskID"`
		Progress float32 `json:"progress"`
		Rate     int64   `json:"rate"`
		State    bool    `json:"state"`
		Time     int64   `json:"time"`
	}

	setting.UploadTaskIDMap.Range(func(k, v interface{}) bool {
		gress := &prog{
			State: false,
		}
		hash := v.(string)
		gress.TaskID = k.(string)
		if p, ok := event.ProgressMap.Load(hash); ok {
			pross := p.(float32)
			if pross > 100 {
				pross = 100
			}
			gress.Progress = pross
			gress.State = true
		}
		// utils.DebugLog("gress.Progress", gress.Progress)
		// utils.DebugLog("f>>>>>>>>>>>>>>>>>>>>", hash)
		gress.Rate = 0
		client.UpConnMap.Range(func(k, v interface{}) bool {
			if strings.HasPrefix(k.(string), hash) {
				vconn := v.(*cf.ClientConn)
				w := vconn.GetSecondWriteFlow()
				gress.Rate += w
			}
			return true
		})
		gress.Time = time.Now().Unix()
		mu.Lock()
		upMap[gress.TaskID] = gress
		mu.Unlock()
		return true
	})

	setting.DownloadTaskIDMap.Range(func(k, v interface{}) bool {
		gress := &prog{
			State: true,
		}
		hash := v.(string)
		gress.TaskID = k.(string)
		if val, ok := setting.DownloadProgressMap.Load(hash); ok {
			gress.Progress = val.(float32)
			if val.(float32) > 100 {
				gress.Progress = 100
			}
		}
		// if file.CheckFilePathEx(hash, k.fileName, k.savePath) {
		// 	utils.DebugLog("file downloaded")
		// 	gress.Progress = 100
		// 	gress.Rate = 0
		// 	gress.State = true
		// }
		gress.Rate = 0
		client.DownloadConnMap.Range(func(k, v interface{}) bool {
			if strings.HasPrefix(k.(string), hash) {
				vconn := v.(*cf.ClientConn)
				re := vconn.GetSecondReadFlow()
				gress.Rate += re
			}
			return true
		})
		gress.Time = time.Now().Unix()
		mu.Lock()
		downMap[gress.TaskID] = gress
		mu.Unlock()
		return true
	})

	mu.Lock()
	m["upList"] = upMap
	m["downList"] = downMap
	mu.Unlock()
	data, err := json.Marshal(m)
	if err != nil {
		utils.ErrorLog("json encode error", err)
	}
	if conn == nil {
		return errors.New("conn closed")
	}
	_, err = conn.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// SocketRead SocketRead
func SocketRead(conn net.Conn) error {
	utils.DebugLog("read")
	buffer := make([]byte, utils.MsgHeaderLen)
	n, err := conn.Read(buffer)
	utils.DebugLog("size:", n)
	if err != nil {
		utils.ErrorLog("read error:", err)
		conn.Close()
		return err
	}
	return nil
}
