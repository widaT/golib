package memstat

import (
	"syscall"
	"runtime"
)

type MemStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
	Self uint64 `json:"self"`
}

func MemStat() MemStatus {
	//自身占用
	memStat := new(runtime.MemStats)
	runtime.ReadMemStats(memStat)
	mem := MemStatus{}
	mem.Self = memStat.Alloc

	//系统占用,仅linux/mac下有效
	//system memory usage
	sysInfo := new(syscall.Sysinfo_t)
	err := syscall.Sysinfo(sysInfo)
	if err == nil {
		mem.All = sysInfo.Totalram * uint64(syscall.Getpagesize())
		mem.Free = sysInfo.Freeram * uint64(syscall.Getpagesize())
		mem.Used = mem.All - mem.Free
	}
	return mem
}