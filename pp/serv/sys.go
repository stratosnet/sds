package serv

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"time"

	"github.com/alex023/clock"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/stratosnet/sds/utils/environment"

	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/pp/p2pserver"
	"github.com/stratosnet/sds/pp/setting"
)

var myClock = clock.NewClock()
var job clock.Job

var dumpClock = clock.NewClock()
var dumpJob clock.Job

var lastGcCount int64
var consecutiveGcOverTriggered int

const (
	INVALID_MEM_LIMIT_FOR_READING  = -1
	CHECK_GCSTATS_INTERVAL         = 30 * time.Second
	THRESHOLD_GC_OVER_TRIGGERED    = 30
	THRESHOLD_NUM_CONSEC_STAY_HIGH = 3
)

func SetSoftMemoryCap() {
	if environment.IsDev() {
		debug.SetMemoryLimit(setting.SoftRamLimitDev)
	} else {
		debug.SetMemoryLimit(setting.SoftRamLimit)
	}
	utils.DebugLog("the ram limit is set to ", debug.SetMemoryLimit(INVALID_MEM_LIMIT_FOR_READING))
}

func CheckGCStats() func() {
	return func() {
		gcStats := &debug.GCStats{}
		debug.ReadGCStats(gcStats)
		if gcStats.NumGC-lastGcCount > THRESHOLD_GC_OVER_TRIGGERED {
			consecutiveGcOverTriggered++
			if consecutiveGcOverTriggered >= THRESHOLD_NUM_CONSEC_STAY_HIGH {
				consecutiveGcOverTriggered = 0
				utils.FatalLogfAndTerminate("Fatal error: RAM usage exceeds the limit for too long")
			}
		} else {
			consecutiveGcOverTriggered = 0
		}

		lastGcCount = gcStats.NumGC
		utils.DebugLogf("GC Stats: last seen(%v), number of gc(%v), ", gcStats.LastGC.Unix(), gcStats.NumGC)
		myClock.AddJobWithInterval(CHECK_GCSTATS_INTERVAL, CheckGCStats())
	}
}

func StartDumpTrafficLog(ctx context.Context) {
	logJobInterval := setting.Config.Traffic.LogInterval
	dumpJob, _ = dumpClock.AddJobRepeat(time.Second*time.Duration(logJobInterval), 0, dumpTrafficLog(ctx))
}

func dumpTrafficLog(ctx context.Context) func() {
	return func() {
		v, _ := mem.VirtualMemory()
		c, _ := cpu.Info()
		cc, _ := cpu.Percent(time.Second, false)
		d, _ := utils.GetDiskUsage(setting.Config.Home.StoragePath)

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
		d.Total = setting.GetDiskSizeSoftCap(d.Total)
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

		if ps := p2pserver.GetP2pServer(ctx); ps != nil {
			if setting.IsPP && ps.GetP2pServer() != nil {
				serverInbound = ps.GetP2pServer().GetInboundAndReset()
				serverOutbound = ps.GetP2pServer().GetOutboundAndReset()
			}
			conn := ps.GetSpConn()
			if conn != nil {
				clientInbound += conn.GetInboundAndReset()
				clientOutbound += conn.GetOutboundAndReset()
			}

			ps.RangeCachedConn("upload#", func(k, v interface{}) bool {
				vconn := v.(*cf.ClientConn)
				in := vconn.GetInboundAndReset()
				clientInbound += in
				out := vconn.GetOutboundAndReset()
				clientOutbound += out
				return true
			})

			ps.RangeCachedConn("download#", func(k, v interface{}) bool {
				vconn := v.(*cf.ClientConn)
				in := vconn.GetInboundAndReset()
				clientInbound += in
				out := vconn.GetOutboundAndReset()
				clientOutbound += out
				return true
			})
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

		trafficInfo.TimeStamp = time.Now().String()[:19]
		TrafficInfoToMonitorClient(trafficInfo)
	}
}

func StopDumpTrafficLog() {
	if dumpJob != nil {
		utils.Log("Stopping DumpTrafficLog Job......")
		dumpJob.Cancel()
	}
}

// ShowMonitor
func ShowMonitor(ctx context.Context) {
	job, _ = myClock.AddJobRepeat(time.Second*time.Duration(setting.Config.Traffic.LogInterval), 0, monitor(ctx))
}

func monitor(ctx context.Context) func() {
	return func() {
		v, _ := mem.VirtualMemory()
		c, _ := cpu.Info()
		cc, _ := cpu.Percent(time.Second, false)
		d, _ := utils.GetDiskUsage(setting.Config.Home.StoragePath)

		d.Total = setting.GetDiskSizeSoftCap(d.Total)
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
		if ps := p2pserver.GetP2pServer(ctx); ps != nil {
			if setting.IsPP && ps.GetP2pServer() != nil {
				r += ps.GetP2pServer().GetSecondReadFlow()
				w += ps.GetP2pServer().GetSecondWriteFlow()
			}
			conn := ps.GetSpConn()
			if conn != nil {
				r += conn.GetSecondReadFlow()
				w += conn.GetSecondWriteFlow()
			}

			ps.RangeCachedConn("upload#", func(k, v interface{}) bool {
				vconn := v.(*cf.ClientConn)
				r1 := vconn.GetSecondReadFlow()
				r += r1
				w1 := vconn.GetSecondWriteFlow()
				w += w1
				return true
			})

			ps.RangeCachedConn("download#", func(k, v interface{}) bool {
				vconn := v.(*cf.ClientConn)
				r1 := vconn.GetSecondReadFlow()
				r += r1
				w1 := vconn.GetSecondWriteFlow()
				w += w1
				return true
			})
		}

		utils.Logf("        Upload      : %f MB/s ", float64(w)/1024/1024)
		utils.Logf("        Download    : %f MB/s ", float64(r)/1024/1024)
		utils.Logf("__________________________________________________________________________")
	}
}

// StopMonitor
func StopMonitor() {
	if job != nil {
		job.Cancel()
	}
}
