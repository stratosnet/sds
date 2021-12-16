package serv

import (
	"fmt"
	"time"

	"github.com/alex023/clock"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
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
	//trafficLogInfo = core.InitTrafficLogInfo()
	logger := utils.NewTrafficLogger("./tmp/logs/stdout.log", false, true)
	logger.SetLogLevel(utils.Info)

	logJobInterval := setting.Config.TrafficLogInterval
	dumpJob, _ = dumpClock.AddJobRepeat(time.Second*time.Duration(logJobInterval), 0, dumpTrafficLog)
}

func dumpTrafficLog() {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Info()
	cc, _ := cpu.Percent(time.Second, false)
	d, _ := disk.Usage("/")

	//Memory
	memTotal := float64(v.Total) / 1024 / 1024    // MB
	memFree := float64(v.Available) / 1024 / 1024 // MB
	memUsed := float64(v.Used) / 1024 / 1024      // MB
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
	hdTotal := float64(d.Total) / 1024 / 1024 / 1024    // GB
	hdFree := float64(d.Free) / 1024 / 1024 / 1024      // GB
	hdUsed := float64(d.Used) / 1024 / 1024 / 1024      // GB
	hdUsedPercent := d.UsedPercent / 1024 / 1024 / 1024 // GB
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

	trafficInbound := float64(clientInbound+serverInbound) / 1024 / 1024    // MB
	trafficOutbound := float64(clientOutbound+serverOutbound) / 1024 / 1024 // MB
	trafficInfo := NewTrafficInfo(trafficInbound, trafficOutbound)

	trafficDumpInfo := NewTrafficDumpInfo(memInfo, cpuInfo, hdInfo, trafficInfo)
	bDumpInfo, err := codec.Cdc.MarshalJSON(trafficDumpInfo)
	if err != nil {
		utils.Log(err)
		return
	}

	utils.DumpTraffic(bDumpInfo)
}

func StopDumpTrafficLog() {
	if dumpJob != nil {
		dumpJob.Cancel()
	}
}

// ShowMonitor
func ShowMonitor() {
	job, _ = myClock.AddJobRepeat(time.Second*2, 0, monitor)
}

//
func monitor() {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Info()
	cc, _ := cpu.Percent(time.Second, false)
	d, _ := disk.Usage("/")
	// n, _ := host.Info()
	// nv, _ := net.IOCounters(true)
	// boottime, _ := host.BootTime()
	// btime := time.Unix(int64(boottime), 0).Format("2006-01-02 15:04:05")
	fmt.Printf("__________________________________________________________________________\n")
	fmt.Printf("        Mem         : %v MB  Free: %v MB Used:%v Usage:%f%%\n", v.Total/1024/1024, v.Available/1024/1024, v.Used/1024/1024, v.UsedPercent)
	if len(c) > 1 {
		for _, sub_cpu := range c {
			modelname := sub_cpu.ModelName
			cores := sub_cpu.Cores
			fmt.Printf("        CPU          : %v   %v cores \n", modelname, cores)
		}
	} else {
		sub_cpu := c[0]
		modelname := sub_cpu.ModelName
		cores := sub_cpu.Cores
		fmt.Printf("        CPU         : %v   %v cores \n", modelname, cores)

	}
	fmt.Printf("        CPU Used    : %f%% \n", cc[0])
	// fmt.Printf("        Network     : %v bytes / %v bytes\n", nv[0].BytesRxecv, nv[0].BytesSent)
	// fmt.Printf("        SystemBoot:%v\n", btime)
	fmt.Printf("        HD          : %v GB  Free: %v GB Usage:%f%%\n", d.Total/1024/1024/1024, d.Free/1024/1024/1024, d.UsedPercent)
	// fmt.Printf("        OS        : %v(%v)   %v  \n", n.Platform, n.PlatformFamily, n.PlatformVersion)
	// fmt.Printf("        Hostname  : %v  \n", n.Hostname)
	r := int64(0)
	w := int64(0)
	if setting.IsPP && peers.GetPPServer() != nil {
		r = peers.GetPPServer().Server.GetSecondReadFlow()
		w = peers.GetPPServer().Server.GetSecondWriteFlow()
		fmt.Printf("        Upload      : %f MB/s \n", float64(w)/1024/1024)
		fmt.Printf("        Download    : %f MB/s \n", float64(r)/1024/1024)
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
		fmt.Printf("        Upload      : %f MB/s \n", float64(w)/1024/1024)
		fmt.Printf("        Download    : %f MB/s \n", float64(r)/1024/1024)
	} else if client.SPConn != nil {
		r = client.SPConn.GetSecondReadFlow()
		w = client.SPConn.GetSecondWriteFlow()
		fmt.Printf("        Upload      : %f MB/s \n", float64(w)/1024/1024)
		fmt.Printf("        Download    : %f MB/s \n", float64(r)/1024/1024)
	} else {
		fmt.Printf("        Upload      : %f MB/s \n", float64(w)/1024/1024)
		fmt.Printf("        Download    : %f MB/s \n", float64(r)/1024/1024)
	}

	fmt.Printf("__________________________________________________________________________\n")

}

// StopMonitor
func StopMonitor() {
	if job != nil {
		job.Cancel()
	}
}
