package core

type WritePacketCostTime struct {
	PacketId int64
	CostTime int64
}

var CostTimeCh = make(chan WritePacketCostTime, 10000)
