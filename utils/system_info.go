package utils

import (
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
	MemorySize uint64
	OSInfo     string
	CPUInfo    string
	MacAddress string
}

// GetSysInfo
func GetSysInfo() *SysInfo {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Info()
	d, _ := disk.Usage("/")
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
		DiskSize:   d.Total,
		MemorySize: v.Total,
		OSInfo:     n.Platform + " " + n.PlatformFamily + " " + n.PlatformVersion,
		CPUInfo:    cupInfo,
		MacAddress: macAddress,
	}
	DebugLog("sysInfo = ", sys)
	return sys
}
