package net

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/stratosnet/sds/utils"
	"net/http"
	"strings"
	"sync"
)

var (
	subscriberID *utils.AtomicInt64
	upgrader     = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func init() {
	subscriberID = utils.CreateAtomicInt64(0)
}

type SubscriptionServer struct {
	httpServer  *http.Server
	server      *Server
	subscribers *sync.Map
}

type subscriber struct {
	msgChan      chan []byte
	once         *sync.Once
	server       *SubscriptionServer
	subscriberID int64
	topics       []string
	ws           *websocket.Conn
}

func NewSubscriptionServer(server *Server) *SubscriptionServer {
	return &SubscriptionServer{
		server:      server,
		subscribers: &sync.Map{},
	}
}

// Start starts an http server that will create websocket connections for subscribing to events happening on the SP node.
func (s *SubscriptionServer) Start() {
	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebsocket(w, r)
	})

	go func() {
		addr := s.server.Conf.Net.Host + ":" + s.server.Conf.Net.WebsocketPort
		utils.Log("Starting subscriptions websocket server at " + addr)
		server := &http.Server{Addr: addr}
		s.httpServer = server
		err := server.ListenAndServe()
		if err != nil {
			utils.ErrorLog(errors.New("couldn't serve http server for websockets: " + err.Error()))
			panic(err)
		}
	}()
}

func (s *SubscriptionServer) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		utils.ErrorLog(errors.New("couldn't upgrade websocket connection: " + err.Error()))
		return
	}

	topics := r.Header.Get("topics")
	if topics == "" {
		utils.ErrorLog(errors.New("can't subscribe without providing a list of topics"))
		return
	}

	sub := &subscriber{
		msgChan:      make(chan []byte),
		once:         &sync.Once{},
		server:       s,
		subscriberID: subscriberID.IncrementAndGetNew(),
		topics:       strings.Split(topics, ","),
		ws:           ws,
	}
	s.subscribers.Store(sub.subscriberID, sub)
	utils.Log(fmt.Sprintf("adding subscriber for topics %v", sub.topics))
	go sub.readerLoop()
	go sub.writerLoop()
}

func (sub *subscriber) readerLoop() {
	defer sub.close()
	for {
		_, _, err := sub.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				utils.ErrorLog(err)
			}
			break
		}
	}
}

func (sub *subscriber) writerLoop() {
	defer sub.close()
	for {
		select {
		case message, ok := <-sub.msgChan:
			if !ok {
				// The channel was closed
				_ = sub.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			utils.Log(fmt.Sprintf("writing %v bytes to websocket", len(message)))
			err := sub.ws.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				utils.ErrorLog(errors.New("couldn't send binary message through websocket: " + err.Error()))
			}
		}
	}
}

func (sub *subscriber) close() {
	sub.once.Do(func() {
		_ = sub.ws.Close()
		sub.server.subscribers.Delete(sub.subscriberID)
	})
}

func (s *SubscriptionServer) Close() {
	s.subscribers.Range(func(k, v interface{}) bool {
		sub := v.(*subscriber)
		sub.close()
		return true
	})

	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
}

func (s *SubscriptionServer) Broadcast(topic string, message []byte) {
	utils.Log(fmt.Sprintf("calling Broadcast for %v bytes to topic %v", len(message), topic))
	s.subscribers.Range(func(k, v interface{}) bool {
		sub := v.(*subscriber)
		utils.Log(fmt.Sprintf("Broadcasting to subscriber"))
		for _, top := range sub.topics {
			if top == topic {
				utils.Log(fmt.Sprintf("topic match"))
				sub.msgChan <- message
			}
		}
		return true
	})
}
