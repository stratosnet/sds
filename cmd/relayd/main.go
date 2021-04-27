package main

import (
	"context"
	"fmt"
	tmhttp "github.com/tendermint/tendermint/rpc/client/http"
	"os"
	"os/signal"
	"syscall"
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

	client, err := tmhttp.New("tcp://"+scWebsocketUrl, "/websocket")
	if err != nil {
		fmt.Println("Failed to create client: " + err.Error())
		return
	}
	err = client.Start()
	if err != nil {
		fmt.Println("Failed to start client: " + err.Error())
		return
	}
	defer client.Stop()

	query := "tm.event = 'NewBlock'"
	out, err := client.Subscribe(context.Background(), "test-relay", query, 1000)
	if err != nil {
		fmt.Println("Failed to subscribe to query: " + err.Error())
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Successfully subscribed. Waiting for messages...")
	for {
		select {
		case result := <-out:
			fmt.Println("Received a new message from stratos-chain!")
			// TODO: handle message
			fmt.Println(result)
			/*
				var responses []types.RPCResponse
				if err = json.Unmarshal(data, &responses); err != nil {
					var response types.RPCResponse
					if err = json.Unmarshal(data, &response); err != nil {
						c.Logger.Error("failed to parse response", "err", err, "data", string(data))
						continue
					}
					responses = []types.RPCResponse{response}
				}
				logger.Info("got tx",
					"index", result.Data.(tmtypes.EventDataTx).Index)
			*/
		case <-quit:
			os.Exit(0)
		}
	}

	/*
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

			var responses []types.RPCResponse
			if err = json.Unmarshal(data, &responses); err != nil {
				var response types.RPCResponse
				if err = json.Unmarshal(data, &response); err != nil {
					c.Logger.Error("failed to parse response", "err", err, "data", string(data))
					continue
				}
				responses = []types.RPCResponse{response}
			}
		}
	*/

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
