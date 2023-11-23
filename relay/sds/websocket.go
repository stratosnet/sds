package sds

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/stratosnet/framework/utils"
)

const (
	TypeBroadcast = "broadcast"
)

func DialWebsocket(addr string, topics []string) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(addr, http.Header{"topics": topics})
	if err != nil {
		utils.ErrorLog(err)
		return nil
	}

	return ws
}
