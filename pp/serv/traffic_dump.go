package serv

type MemInfo struct {
	MemTotal       uint64  `json:"mem_total"`
	MemFree        uint64  `json:"mem_free"`
	MemUsed        uint64  `json:"mem_used"`
	MemUsedPercent float64 `json:"mem_used_percent"`
}

func NewMemInfo(memTotal uint64, memFree uint64, memUsed uint64, memUsedPercent float64) MemInfo {
	return MemInfo{
		MemTotal:       memTotal,
		MemFree:        memFree,
		MemUsed:        memUsed,
		MemUsedPercent: memUsedPercent,
	}
}

type SingleCpuInfo struct {
	CpuModelName string `json:"cpu_model_name"`
	CpuCores     int32  `json:"cpu_cores"`
}

func NewSingleCpuInfo(cpuModelName string, cpuCores int32) SingleCpuInfo {
	return SingleCpuInfo{
		CpuModelName: cpuModelName,
		CpuCores:     cpuCores,
	}
}

type CpuInfo struct {
	CpuInfos       []SingleCpuInfo `json:"cpu_infos"`
	CpuUsedPercent float64         `json:"cpu_used_percent"`
}

func NewCpuInfo(cpuInfos []SingleCpuInfo, cpuUsedPercent float64) CpuInfo {
	return CpuInfo{
		CpuInfos:       cpuInfos,
		CpuUsedPercent: cpuUsedPercent,
	}
}

type HdInfo struct {
	HdTotal       uint64  `json:"hd_total"`
	HdFree        uint64  `json:"hd_free"`
	HdUsed        uint64  `json:"hd_used"`
	HdUsedPercent float64 `json:"hd_used_percent"`
}

func NewHdInfo(hdTotal uint64, hdFree uint64, hdUsed uint64, hdUsedPercent float64) HdInfo {
	return HdInfo{
		HdTotal:       hdTotal,
		HdFree:        hdFree,
		HdUsed:        hdUsed,
		HdUsedPercent: hdUsedPercent,
	}
}

type TrafficInfo struct {
	TrafficInbound  uint64 `json:"traffic_inbound"`
	TrafficOutbound uint64 `json:"traffic_outbound"`
}

func NewTrafficInfo(trafficInbound uint64, trafficOutbound uint64) TrafficInfo {
	return TrafficInfo{
		TrafficInbound:  trafficInbound,
		TrafficOutbound: trafficOutbound,
	}
}

type TrafficDumpInfo struct {
	MemInfo     MemInfo     `json:"mem_info"`
	CpuInfo     CpuInfo     `json:"cpu_info"`
	HdInfo      HdInfo      `json:"hd_info"`
	TrafficInfo TrafficInfo `json:"traffic_info"`
}

func NewTrafficDumpInfo(memInfo MemInfo, cpuInfo CpuInfo, hdInfo HdInfo, trafficInfo TrafficInfo) TrafficDumpInfo {
	return TrafficDumpInfo{
		MemInfo:     memInfo,
		CpuInfo:     cpuInfo,
		HdInfo:      hdInfo,
		TrafficInfo: trafficInfo,
	}
}
