package serv

type MemInfo struct {
	MemTotal       float64 `json:"mem_total"`
	MemFree        float64 `json:"mem_free"`
	MemUsed        float64 `json:"mem_used"`
	MemUsedPercent float64 `json:"mem_used_percent"`
}

func NewMemInfo(memTotal float64, memFree float64, memUsed float64, memUsedPercent float64) MemInfo {
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
	HdTotal       float64 `json:"hd_total"`
	HdFree        float64 `json:"hd_free"`
	HdUsed        float64 `json:"hd_used"`
	HdUsedPercent float64 `json:"hd_used_percent"`
}

func NewHdInfo(hdTotal float64, hdFree float64, hdUsed float64, hdUsedPercent float64) HdInfo {
	return HdInfo{
		HdTotal:       hdTotal,
		HdFree:        hdFree,
		HdUsed:        hdUsed,
		HdUsedPercent: hdUsedPercent,
	}
}

type TrafficInfo struct {
	TrafficInbound  float64 `json:"traffic_inbound"`
	TrafficOutbound float64 `json:"traffic_outbound"`
}

func NewTrafficInfo(trafficInbound float64, trafficOutbound float64) TrafficInfo {
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
