package utils

import (
	"path/filepath"
	"strconv"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

// SysInfo
type SysInfo struct {
	DiskSize   uint64
	FreeDisk   uint64
	MemorySize uint64
	OSInfo     string
	CPUInfo    string
	MacAddress string
}

// GetSysInfo
func GetSysInfo(diskPath string) *SysInfo {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Info()
	n, _ := host.Info()
	nv, _ := net.Interfaces()
	cupInfo := ""
	if len(c) > 1 {
		for _, subCPU := range c {
			modelname := subCPU.ModelName
			cores := subCPU.Cores
			cupInfo = modelname + " " + strconv.Itoa(int(cores)) + "	"
		}
	} else {
		subCPU := c[0]
		modelname := subCPU.ModelName
		cores := subCPU.Cores
		cupInfo = modelname + " " + strconv.Itoa(int(cores))
	}
	macAddress := ""
	for _, netstat := range nv {
		if netstat.HardwareAddr != "" {
			macAddress = netstat.HardwareAddr
		}
	}

	sys := &SysInfo{
		MemorySize: v.Total,
		OSInfo:     n.Platform + " " + n.PlatformFamily + " " + n.PlatformVersion,
		CPUInfo:    cupInfo,
		MacAddress: macAddress,
	}
	defer DebugLog("sysInfo = ", sys)

	diskStats, err := GetDiskUsage(diskPath)
	if err != nil {
		ErrorLog("Can't fetch disk usage statistics", err)
		return sys
	}

	sys.DiskSize = diskStats.Total
	sys.FreeDisk = diskStats.Free
	return sys
}

func GetDiskUsage(path string) (*disk.UsageStat, error) {

	volume := filepath.Dir(path)

	return disk.Usage(volume)
}
