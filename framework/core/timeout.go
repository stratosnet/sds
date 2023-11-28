package core

import (
	"context"
	"sync"
	"time"

	"github.com/stratosnet/sds/sds-msg/header"

	"github.com/stratosnet/sds/framework/msg"
)

var (
	TimeoutRegistry [header.NUMBER_MESSAGE_TYPES]TimeoutHandler
	TimoutMap       = newTimeoutMap()
)

type TimeoutHandler interface {
	Handle(ctx context.Context, message *msg.RelayMsgBuf)
	GetDuration() time.Duration
	GetTimeoutMsg(reqMessage *msg.RelayMsgBuf) *msg.RelayMsgBuf
	CanDelete(rspMessage *msg.RelayMsgBuf) bool
}

func RegisterTimeoutHandler(msgType header.MsgType, handler TimeoutHandler) {
	TimeoutRegistry[msgType.Id] = handler
}

type timeoutMap struct {
	myMap *sync.Map
}

type MyValue struct {
	message   *msg.RelayMsgBuf
	deletedCh chan bool
	handler   TimeoutHandler
}

func newTimeoutMap() *timeoutMap {
	return &timeoutMap{
		myMap: &sync.Map{},
	}
}

func (m *timeoutMap) Store(ctx context.Context, reqId int64, reqMsg *msg.RelayMsgBuf) {
	handler := TimeoutRegistry[reqMsg.MSGHead.Cmd]
	if handler == nil {
		return
	}

	m.Delete(reqId)
	deletedCh := make(chan bool, 1)
	m.myMap.Store(reqId, &MyValue{
		message:   handler.GetTimeoutMsg(reqMsg),
		deletedCh: deletedCh,
		handler:   handler,
	})

	go func() {
		select {
		case deleted := <-deletedCh:
			if deleted {
				return
			}
		case <-time.After(handler.GetDuration()):
			go handler.Handle(ctx, reqMsg)
		}
		m.myMap.Delete(reqId)
	}()
}

func (m *timeoutMap) Load(key interface{}) (interface{}, bool) {
	if value, ok := m.myMap.Load(key); ok {
		myValue := value.(*MyValue)
		return myValue.message, true
	} else {
		return nil, false
	}
}

func (m *timeoutMap) HasKey(key interface{}) bool {
	_, ok := m.myMap.Load(key)
	return ok
}

func (m *timeoutMap) Delete(key interface{}) {
	if value, ok := m.myMap.Load(key); ok {
		myValue := value.(*MyValue)
		m.myMap.Delete(key)
		myValue.deletedCh <- true
	}
}

func (m *timeoutMap) DeleteByRspMsg(rspMsg *msg.RelayMsgBuf) {
	reqId := rspMsg.MSGHead.ReqId
	if value, ok := m.myMap.Load(reqId); ok {
		myValue := value.(*MyValue)
		if myValue.handler.CanDelete(rspMsg) {
			m.Delete(reqId)
		}
	}
}
