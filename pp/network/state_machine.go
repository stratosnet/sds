package network

import (
	"context"

	"github.com/stratosnet/sds/framework/utils"
)

const (
	EVENT_SP_NO_PP_IN_STORE uint64 = iota
	EVENT_RCV_RSP_REGISTER_NEW_PP
	EVENT_RCV_STATUS_INACTIVE
	EVENT_RCV_STATUS_SUSPEND
	EVENT_RCV_RSP_ACTIVATED
	EVENT_RCV_STATE_OFFLINE
	EVENT_START_MINING
	EVENT_RCV_MINING_NOT_STARTED
	EVENT_MAINTANENCE_STOP
	EVENT_MAINTANENCE_START
	EVENT_CONN_RECONN
	EVENT_RCV_RSP_FIRST_NODE_STATUS
	EVENT_RCV_SUSPENDED_STATE
	EVENT_GET_SP_LIST
	NUMBER_EVENTS
)

const (
	STATE_INIT uint64 = iota
	STATE_GOT_SP_LIST
	STATE_NOT_CREATED
	STATE_NOT_ACTIVATED
	STATE_NOT_REGISTERED
	STATE_REGISTERING
	STATE_REGISTERED
	STATE_MAINTANENCE
	STATE_SUSPENDED
	NUMBER_STATE
)

var (
	e_list = [NUMBER_EVENTS]utils.Event{
		{Id: EVENT_SP_NO_PP_IN_STORE, Name: "EVENT_SP_NO_PP_IN_STORE"},
		{Id: EVENT_RCV_RSP_REGISTER_NEW_PP, Name: "EVENT_RCV_RSP_REGISTER_NEW_PP"},
		{Id: EVENT_RCV_STATUS_INACTIVE, Name: "EVENT_RCV_STATUS_INACTIVE"},
		{Id: EVENT_RCV_STATUS_SUSPEND, Name: "EVENT_RCV_STATUS_SUSPEND"},
		{Id: EVENT_RCV_RSP_ACTIVATED, Name: "EVENT_RCV_RSP_ACTIVATED"},
		{Id: EVENT_RCV_STATE_OFFLINE, Name: "EVENT_RCV_STATE_OFFLINE"},
		{Id: EVENT_START_MINING, Name: "EVENT_START_MINING"},
		{Id: EVENT_RCV_MINING_NOT_STARTED, Name: "EVENT_RCV_MINING_NOT_STARTED"},
		{Id: EVENT_MAINTANENCE_STOP, Name: "EVENT_MAINTANENCE_STOP"},
		{Id: EVENT_MAINTANENCE_START, Name: "EVENT_MAINTANENCE_START"},
		{Id: EVENT_CONN_RECONN, Name: "EVENT_CONN_RECONN"},
		{Id: EVENT_RCV_RSP_FIRST_NODE_STATUS, Name: "EVENT_RCV_RSP_FIRST_NODE_STATUS"},
		{Id: EVENT_RCV_SUSPENDED_STATE, Name: "EVENT_RCV_SUSPENDED_STATE"},
		{Id: EVENT_GET_SP_LIST, Name: "EVENT_GET_SP_LIST"},
	}
	s_list = [NUMBER_STATE]utils.State{
		{Id: STATE_INIT, Name: "STATE_INIT"},
		{Id: STATE_GOT_SP_LIST, Name: "STATE_GOT_SP_LIST"},
		{Id: STATE_NOT_CREATED, Name: "STATE_NOT_CREATED"},
		{Id: STATE_NOT_ACTIVATED, Name: "STATE_NOT_ACTIVATED"},
		{Id: STATE_NOT_REGISTERED, Name: "STATE_NOT_REGISTERED"},
		{Id: STATE_REGISTERING, Name: "STATE_REGISTERING"},
		{Id: STATE_REGISTERED, Name: "STATE_REGISTERED"},
		{Id: STATE_MAINTANENCE, Name: "STATE_MAINTANENCE"},
		{Id: STATE_SUSPENDED, Name: "STATE_SUSPENDED"},
	}
)

type fsmTableItem struct {
	oldState       uint64
	eventId        uint64
	transitionItem utils.TransitionItem
}

func (n *Network) InitFsm() {
	var sl []utils.State
	for s := range s_list {
		sl = append(sl, s_list[s])
	}

	var el []utils.Event
	for e := range e_list {
		el = append(el, e_list[e])
	}
	var fsmTable = []fsmTableItem{
		{STATE_INIT, EVENT_GET_SP_LIST, utils.TransitionItem{NewState: STATE_GOT_SP_LIST, Action: n.GetPPStatusFromSP}},

		{STATE_GOT_SP_LIST, EVENT_SP_NO_PP_IN_STORE, utils.TransitionItem{NewState: STATE_NOT_CREATED}},
		{STATE_GOT_SP_LIST, EVENT_RCV_STATUS_INACTIVE, utils.TransitionItem{NewState: STATE_NOT_ACTIVATED}},

		{STATE_NOT_CREATED, EVENT_RCV_RSP_REGISTER_NEW_PP, utils.TransitionItem{NewState: STATE_NOT_ACTIVATED}},

		{STATE_NOT_ACTIVATED, EVENT_RCV_RSP_ACTIVATED, utils.TransitionItem{NewState: STATE_NOT_REGISTERED}},

		{STATE_NOT_REGISTERED, EVENT_START_MINING, utils.TransitionItem{NewState: STATE_REGISTERING, Action: n.StartRegisterToSp}},

		{STATE_REGISTERING, EVENT_RCV_RSP_FIRST_NODE_STATUS, utils.TransitionItem{NewState: STATE_REGISTERED}},
		{STATE_REGISTERING, EVENT_RCV_SUSPENDED_STATE, utils.TransitionItem{NewState: STATE_SUSPENDED}},

		{STATE_REGISTERED, EVENT_CONN_RECONN, utils.TransitionItem{NewState: STATE_REGISTERING, Action: n.StartRegisterToSp}},
		{STATE_REGISTERED, EVENT_RCV_SUSPENDED_STATE, utils.TransitionItem{NewState: STATE_SUSPENDED}},
		{STATE_REGISTERED, EVENT_MAINTANENCE_START, utils.TransitionItem{NewState: STATE_MAINTANENCE}},
		{STATE_REGISTERED, EVENT_RCV_STATUS_SUSPEND, utils.TransitionItem{NewState: STATE_SUSPENDED}},

		{STATE_MAINTANENCE, EVENT_MAINTANENCE_STOP, utils.TransitionItem{NewState: STATE_REGISTERING}},

		{STATE_SUSPENDED, EVENT_START_MINING, utils.TransitionItem{NewState: STATE_REGISTERING, Action: n.StartRegisterToSp}},
		{STATE_SUSPENDED, EVENT_RCV_MINING_NOT_STARTED, utils.TransitionItem{NewState: STATE_NOT_REGISTERED}},
	}

	_ = n.fsm.InitFsm(sl, el, func(fsm *utils.Fsm) {
		// init to self transition entries
		for row := range s_list {
			var itemslist []utils.TransitionItem
			for range e_list {
				itemslist = append(itemslist, utils.TransitionItem{NewState: sl[row].Id})
			}
			fsm.StateTransTable = append(fsm.StateTransTable, itemslist)
		}
		// copy the items into state transition table
		for row := range fsmTable {
			s := fsmTable[row].oldState
			e := fsmTable[row].eventId
			fsm.StateTransTable[s][e] = fsmTable[row].transitionItem
		}
	}, STATE_INIT)
}

func (n *Network) RunFsm(ctx context.Context, eventId uint64) {
	n.fsm.RunFsm(ctx, eventId)
}

func (n *Network) GetStateFromFsm() utils.State {
	state := n.fsm.GetState()
	utils.DebugLog("fsm current state:", state.Name)
	return state
}
