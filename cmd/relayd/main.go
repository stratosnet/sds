package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/stratosnet/sds/cmd/relayd/utils"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
)

func main() {
	scWebsocketUrl := "localhost:26657"

	//sdsClientUrl := "localhost:8888"
	//sdsWebsocketUrl := "localhost:8889"

	// Send a msg to subscribe to stratos-chain
	// Push the info to SP node

	// use utils to talk to stratos-chain and SP/PP
	// subscribe to events on stratos-chain and SP/PP

	// Send RPC message to stratos-chain
	// According to the cosmos-sdk docs, there should be a gRPC service defined for each module, although it is not there yet

	// Subscribe to events from stratos-chain
	// It should be possible to subscribe to events in cosmos-sdk by using websockets at the tendermint layer

	scFullWebsocketUrl := "ws://" + scWebsocketUrl + "/websocket"
	ws, _, err := websocket.DefaultDialer.Dial(scFullWebsocketUrl, http.Header{})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("statos-chain websocket connection created")

	conn := utils.NewRWCConn(ws)
	defer conn.Close()

	codec := jsonrpc.NewClientCodec(conn)
	rpcClient := rpc.NewClientWithCodec(codec)

	// TODO: create appropriate types for args and reply
	/*type Args struct {
		query string
	}*/
	type SubscribeResponse struct {
		error string
	}
	reply := SubscribeResponse{}
	err = rpcClient.Call("subscribe", "tm.event = 'NewBlock'", &reply)
	if err != nil {
		fmt.Println("couldn't call RPC subscribe: " + err.Error())
		return
	}
	fmt.Println("stratos-chain subscription success")
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Println("An error occurred when reading from the stratos-chain websocket: " + err.Error())
				break
			} else {
				fmt.Println("Unknown error when reading message from stratos-chain: " + err.Error())
			}
		}
		fmt.Println("Received a new message from stratos-chain!")
		// TODO: handle message
		fmt.Println(message)
	}

	/*
		// Send message to SP
		sdsClient := sds.NewClient(sdsClientUrl)
		msgToSend := &msg.RelayMsgBuf{
			MSGHead: header.MakeMessageHeader(1, 1, 0, header.ReqGetPPList),
		}
		err = sdsClient.Write(msgToSend)
		if err != nil {
			fmt.Println("Error when sending message to SDS: " + err.Error())
		} else {
			fmt.Println("Sent msg to SDS")
		}

		// Subscribe to events from SP
		fullSdsWebsocketUrl := "ws://" + sdsWebsocketUrl + "/websocket"
		ws = sds.DialWebsocket(fullSdsWebsocketUrl)
		if ws == nil {
			fmt.Println("Couldn't subscribe to SDS websocket")
			return
		}
		defer ws.Close()
		sds.ReaderLoop(ws)
	*/
}
