package sds

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/stratosnet/sds/msg/protos"
	"net/http"
)

func DialWebsocket(addr string) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(addr, http.Header{"topics": []string{"test"}})
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	return ws
}

func ReaderLoop(ws *websocket.Conn) {
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println("received: " + string(data))
		msg := protos.RspGetPPList{}
		proto.Unmarshal(data, &msg)
		fmt.Printf("Received: %v\n", msg)
	}
}
