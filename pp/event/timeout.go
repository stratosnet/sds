package event

import (
	"context"
	"sync"
	"time"

	"github.com/stratosnet/sds/framework/msg"
	"github.com/stratosnet/sds/framework/msg/header"
)

const (
	TYPE_REQ_RSP_TIMER        = 1
	TYPE_RSP_LAST_TOUCH_TIMER = 2
)

var (
	TimeoutRegistry [header.NUMBER_MESSAGE_TYPES]TimeoutHandler
	TimoutMap       = newTimeoutMap()
)

type TimeoutHandler interface {
	TimeoutHandler(ctx context.Context, message *msg.RelayMsgBuf)
	GetDuration() time.Duration
	GetTimeoutMsg(reqMessage *msg.RelayMsgBuf) *msg.RelayMsgBuf
	CanDelete(rspMessage *msg.RelayMsgBuf) bool
	Update(key string) bool
	GetId(message *msg.RelayMsgBuf, isReq bool) string
	GetType() int
}

func RegisterTimeoutHandler(msgType header.MsgType, handler TimeoutHandler) {
	TimeoutRegistry[msgType.Id] = handler
}

type timeoutMap struct {
	myMap *sync.Map
}

type MyValue struct {
	key       any
	message   *msg.RelayMsgBuf
	updatedCh chan bool
	deletedCh chan bool
	handler   TimeoutHandler
}

func newTimeoutMap() *timeoutMap {
	return &timeoutMap{
		myMap: &sync.Map{},
	}
}

func (m *timeoutMap) startTimer(ctx context.Context, t *MyValue, msg *msg.RelayMsgBuf) {
	go func() {
		for {
			select {
			case deleted := <-t.deletedCh:
				if deleted {
					return
				}
			case <-t.updatedCh:
			case <-time.After(t.handler.GetDuration()):
				go t.handler.TimeoutHandler(ctx, msg)
				m.myMap.Delete(t.key)
				return
			}
		}
	}()
}

func (m *timeoutMap) getKeyFromLastTouchType(taskId string) string {
	return "LAST_TOUCH#" + taskId
}

func (m *timeoutMap) OnWrite(ctx context.Context, reqMsg *msg.RelayMsgBuf) {
	handler := TimeoutRegistry[reqMsg.MSGHead.Cmd]
	if handler == nil {
		return
	}
	if handler.GetType() == TYPE_REQ_RSP_TIMER {
		reqId := reqMsg.MSGHead.ReqId
		m.Delete(reqId)
		t := &MyValue{
			key:       reqId,
			message:   handler.GetTimeoutMsg(reqMsg),
			deletedCh: make(chan bool, 1),
			handler:   handler,
		}
		m.myMap.Store(reqId, t)
		m.startTimer(ctx, t, reqMsg)
		return
	}

	taskId := handler.GetId(reqMsg, true)
	key := m.getKeyFromLastTouchType(taskId)
	if _, ok := m.myMap.Load(key); !ok {
		t := &MyValue{
			key:       key,
			message:   handler.GetTimeoutMsg(reqMsg),
			deletedCh: make(chan bool, 1),
			updatedCh: make(chan bool, 1),
			handler:   handler,
		}
		m.myMap.Store(key, t)
		m.startTimer(ctx, t, reqMsg)
	}
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

func (m *timeoutMap) OnHandle(ctx context.Context, rspMsg *msg.RelayMsgBuf) {
	handler := TimeoutRegistry[header.GetReqIdFromRspId(rspMsg.MSGHead.Cmd)]
	if handler == nil {
		return
	}
	taskId := handler.GetId(rspMsg, false)
	key := m.getKeyFromLastTouchType(taskId)
	if value, ok := m.myMap.Load(key); ok {
		if handler.Update(taskId) {
			value.(*MyValue).updatedCh <- true
		}
	}
}
