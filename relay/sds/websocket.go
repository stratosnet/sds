package sds

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

func DialWebsocket(addr string, topics []string) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(addr, http.Header{"topics": topics})
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	return ws
}
