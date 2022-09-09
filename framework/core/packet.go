package core

type WritePacketCostTime struct {
	ReqId    string
	CostTime int64
}

var CostTimeCh = make(chan *WritePacketCostTime)
