package serv

import (
	"encoding/json"
	"time"

	"github.com/alex023/clock"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
	"github.com/stratosnet/sds/utils"
)

var myClock = clock.NewClock()
var job clock.Job

var dumpClock = clock.NewClock()
var dumpJob clock.Job

func StartDumpTrafficLog() {
	logJobInterval := setting.Config.TrafficLogInterval
	dumpJob, _ = dumpClock.AddJobRepeat(time.Second*time.Duration(logJobInterval), 0, dumpTrafficLog)
}

func dumpTrafficLog() {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Info()
	cc, _ := cpu.Percent(time.Second, false)
	d, _ := utils.GetDiskUsage(setting.Config.StorehousePath)

	//Memory
	memTotal := v.Total
	memFree := v.Available
	memUsed := v.Used
	memUsedPercent := v.UsedPercent
	memInfo := NewMemInfo(memTotal, memFree, memUsed, memUsedPercent)

	//CPU
	cpuInfos := make([]SingleCpuInfo, 0)
	if len(c) > 0 {
		for _, sub_cpu := range c {
			cpuModelName := sub_cpu.ModelName
			cpuCores := sub_cpu.Cores
			singleCpuInfo := NewSingleCpuInfo(cpuModelName, cpuCores)
			cpuInfos = append(cpuInfos, singleCpuInfo)
		}
	}
	cpuUsedPercent := cc[0]
	cpuInfo := NewCpuInfo(cpuInfos, cpuUsedPercent)

	//HD
	hdTotal := d.Total
	hdFree := d.Free
	hdUsed := d.Used
	hdUsedPercent := d.UsedPercent
	hdInfo := NewHdInfo(hdTotal, hdFree, hdUsed, hdUsedPercent)

	//Traffic
	serverInbound := int64(0)
	serverOutbound := int64(0)
	clientInbound := int64(0)
	clientOutbound := int64(0)

	if setting.IsPP && peers.GetPPServer() != nil {
		serverInbound = peers.GetPPServer().Server.GetInboundAndReset()
		serverOutbound = peers.GetPPServer().Server.GetOutboundAndReset()
	}
	if client.PPConn != nil {
		clientInbound += client.PPConn.GetInboundAndReset()
		clientOutbound += client.PPConn.GetSecondWriteFlow()
		client.UpConnMap.Range(func(k, v interface{}) bool {
			vconn := v.(*cf.ClientConn)
			in := vconn.GetInboundAndReset()
			clientInbound += in
			out := vconn.GetOutboundAndReset()
			clientOutbound += out
			return true
		})
	}
	if client.SPConn != nil {
		clientInbound += client.SPConn.GetInboundAndReset()
		clientOutbound += client.SPConn.GetOutboundAndReset()
	}

	trafficInbound := uint64(clientInbound + serverInbound)
	trafficOutbound := uint64(clientOutbound + serverOutbound)
	trafficInfo := NewTrafficInfo(trafficInbound, trafficOutbound)

	trafficDumpInfo := NewTrafficDumpInfo(memInfo, cpuInfo, hdInfo, trafficInfo)
	bDumpInfo, err := json.Marshal(trafficDumpInfo)
	if err != nil {
		utils.Log(err)
		return
	}

	utils.DumpTraffic(string(bDumpInfo))
}

func StopDumpTrafficLog() {
	if dumpJob != nil {
		dumpJob.Cancel()
	}
}

// ShowMonitor
func ShowMonitor() {
	job, _ = myClock.AddJobRepeat(time.Second*time.Duration(setting.Config.TrafficLogInterval), 0, monitor)
}

//
func monitor() {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Info()
	cc, _ := cpu.Percent(time.Second, false)
	d, _ := utils.GetDiskUsage(setting.Config.StorehousePath)
	// n, _ := host.Info()
	// nv, _ := net.IOCounters(true)
	// boottime, _ := host.BootTime()
	// btime := time.Unix(int64(boottime), 0).Format("2006-01-02 15:04:05")
	utils.Logf("__________________________________________________________________________")
	utils.Logf("        Mem         : %v MB  Free: %v MB Used:%v Usage:%f%%", v.Total/1024/1024, v.Available/1024/1024, v.Used/1024/1024, v.UsedPercent)
	if len(c) > 1 {
		for _, sub_cpu := range c {
			modelname := sub_cpu.ModelName
			cores := sub_cpu.Cores
			utils.Logf("        CPU          : %v   %v cores ", modelname, cores)
		}
	} else {
		sub_cpu := c[0]
		modelname := sub_cpu.ModelName
		cores := sub_cpu.Cores
		utils.Logf("        CPU         : %v   %v cores ", modelname, cores)

	}
	utils.Logf("        CPU Used    : %f%% ", cc[0])
	// utils.Logf("        Network     : %v bytes / %v bytes", nv[0].BytesRxecv, nv[0].BytesSent)
	// utils.Logf("        SystemBoot:%v", btime)
	utils.Logf("        HD          : %v GB  Free: %v GB Usage:%f%% Path:%s", d.Total/1024/1024/1024, d.Free/1024/1024/1024, d.UsedPercent, d.Path)
	// utils.Logf("        OS        : %v(%v)   %v  ", n.Platform, n.PlatformFamily, n.PlatformVersion)
	// utils.Logf("        Hostname  : %v  ", n.Hostname)
	r := int64(0)
	w := int64(0)
	if setting.IsPP && peers.GetPPServer() != nil {
		r = peers.GetPPServer().Server.GetSecondReadFlow()
		w = peers.GetPPServer().Server.GetSecondWriteFlow()
		utils.Logf("        Upload      : %f MB/s ", float64(w)/1024/1024)
		utils.Logf("        Download    : %f MB/s ", float64(r)/1024/1024)
	} else if client.PPConn != nil {
		r = client.PPConn.GetSecondReadFlow()
		w = client.PPConn.GetSecondWriteFlow()
		client.UpConnMap.Range(func(k, v interface{}) bool {
			vconn := v.(*cf.ClientConn)
			r1 := vconn.GetSecondReadFlow()
			r += r1
			w1 := vconn.GetSecondWriteFlow()
			w += w1
			return true
		})
		utils.Logf("        Upload      : %f MB/s ", float64(w)/1024/1024)
		utils.Logf("        Download    : %f MB/s ", float64(r)/1024/1024)
	} else if client.SPConn != nil {
		r = client.SPConn.GetSecondReadFlow()
		w = client.SPConn.GetSecondWriteFlow()
		utils.Logf("        Upload      : %f MB/s ", float64(w)/1024/1024)
		utils.Logf("        Download    : %f MB/s ", float64(r)/1024/1024)
	} else {
		utils.Logf("        Upload      : %f MB/s ", float64(w)/1024/1024)
		utils.Logf("        Download    : %f MB/s ", float64(r)/1024/1024)
	}

	utils.Logf("__________________________________________________________________________")

}

// StopMonitor
func StopMonitor() {
	if job != nil {
		job.Cancel()
	}
}
