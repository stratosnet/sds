package file

import (
	"io/fs"
	"syscall"
)

func accessTime(stat fs.FileInfo) syscall.Timespec {
	return stat.Sys().(*syscall.Stat_t).Atim
}
