package utils

import (
	"os"
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
	Used       uint64
	MemorySize uint64
	OSInfo     string
	CPUInfo    string
	MacAddress string
}

var DiskMeasureFolder string

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

	sys.DiskSize, sys.Used = GetDiskUsage()
	return sys
}

func WalkSize(path string) (totalSize int64) {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		DebugLog("Failed calculate the size of disk used.")
		return 0
	}

	return
}

func SetDiskMeasureFolder(path string) {
	DiskMeasureFolder = path
}
func GetDiskUsage() (uint64, uint64) {

	if DiskMeasureFolder == "" {
		return 0, 0
	}
	df, err := disk.Usage(DiskMeasureFolder)
	if err != nil {
		df.Total = 0
	}
	used := WalkSize(DiskMeasureFolder)

	return df.Total, uint64(used)
}
