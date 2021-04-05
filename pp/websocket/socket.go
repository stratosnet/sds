package websocket

import (
	"encoding/json"
	"errors"
	"github.com/qsnetwork/sds/framework/client/cf"
	"github.com/qsnetwork/sds/pp/client"
	"github.com/qsnetwork/sds/pp/event"
	"github.com/qsnetwork/sds/pp/setting"
	"github.com/qsnetwork/sds/utils"
	"net"
	"sync"
	"time"
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

	setting.UpLoadTaskIDMap.Range(func(k, v interface{}) bool {
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
		if up, ok := client.UpConnMap.Load(hash); ok {
			vconn := up.(*cf.ClientConn)
			w := vconn.GetSecondWriteFlow()
			gress.Rate = w
			gress.Time = time.Now().Unix()
			mu.Lock()
			upMap[gress.TaskID] = gress
			mu.Unlock()
		} else {
			gress.Rate = 0
			gress.Time = time.Now().Unix()
			mu.Lock()
			upMap[gress.TaskID] = gress
			mu.Unlock()
		}
		return true
	})

	setting.DownLoadTaskIDMap.Range(func(k, v interface{}) bool {
		gress := &prog{
			State: true,
		}
		hash := v.(string)
		gress.TaskID = k.(string)
		if val, ok := setting.DownProssMap.Load(hash); ok {
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
		if c, ok := client.PdownloadPassageway.Load(hash); ok {
			conn := c.(*cf.ClientConn)
			re := conn.GetSecondReadFlow()
			gress.Rate = re
			gress.Time = time.Now().Unix()
		}
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
