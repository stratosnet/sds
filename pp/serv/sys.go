package serv

import (
	"fmt"
	"time"

	"github.com/alex023/clock"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/stratosnet/sds/framework/client/cf"
	"github.com/stratosnet/sds/pp/client"
	"github.com/stratosnet/sds/pp/peers"
	"github.com/stratosnet/sds/pp/setting"
)

var myClock = clock.NewClock()
var job clock.Job

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

