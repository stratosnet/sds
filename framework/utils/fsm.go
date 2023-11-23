package utils

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

type Event struct {
	Id   uint64
	Name string
}

type State struct {
	Id   uint64
	Name string
}

type Action func(ctx context.Context)

type TransitionItem struct {
	NewState uint64
	Action   Action
}
type Fsm struct {
	initStateId     uint64
	stateId         uint64
	StateTransTable [][]TransitionItem
	stateList       []State
	eventList       []Event
	mutex           sync.Mutex
}

func (e Event) string() string {
	return e.Name
}
func (s State) string() string {
	return s.Name
}

func (fsm *Fsm) RunFsm(ctx context.Context, e uint64) {
	fsm.mutex.Lock()
	oldState := fsm.stateList[fsm.stateId]
	item := fsm.StateTransTable[fsm.stateId][e]
	fsm.stateId = item.NewState
	if item.Action != nil {
		item.Action(ctx)
	}
	DebugLogf("RunFsm: S.%v ---E.%v---> S.%v", oldState.string(), fsm.eventList[e].string(), fsm.stateList[fsm.stateId].string())
	fsm.mutex.Unlock()
}

func (fsm *Fsm) GetState() State {
	return fsm.stateList[fsm.stateId]
}

func (fsm *Fsm) InitFsm(stateList []State, eventList []Event, f func(f *Fsm), initStateId uint64) error {
	if len(stateList) == 0 || len(eventList) == 0 {
		return errors.New("invalid number of states or events")
	}
	fsm.stateList = stateList
	fsm.eventList = eventList
	f(fsm)
	fsm.initStateId = initStateId
	return nil
}
