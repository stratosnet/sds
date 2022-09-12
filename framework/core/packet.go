package core

type WritePacketCostTime struct {
	ReqId    int64
	CostTime int64
}

var CostTimeCh = make(chan WritePacketCostTime, 10000)
