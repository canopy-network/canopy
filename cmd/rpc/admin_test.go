package rpc

import (
	"strings"
	"testing"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	gnet "github.com/shirou/gopsutil/net"
)

func TestNewSystemResourceUsageRequiresStats(t *testing.T) {
	pm := &mem.VirtualMemoryStat{}
	d := &disk.UsageStat{}
	cpuTimes := []cpu.TimesStat{{User: 1}}
	cpuPercent := []float64{2}
	ioCounters := []gnet.IOCountersStat{{BytesRecv: 3}}

	tests := []struct {
		name       string
		cpuTimes   []cpu.TimesStat
		cpuPercent []float64
		ioCounters []gnet.IOCountersStat
		want       string
	}{
		{name: "missing cpu percent", cpuTimes: cpuTimes, ioCounters: ioCounters, want: "missing cpu percent stats"},
		{name: "missing cpu times", cpuPercent: cpuPercent, ioCounters: ioCounters, want: "missing cpu time stats"},
		{name: "missing io counters", cpuTimes: cpuTimes, cpuPercent: cpuPercent, want: "missing network io stats"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSystemResourceUsage(pm, tt.cpuTimes, tt.cpuPercent, d, tt.ioCounters)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("newSystemResourceUsage error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestNewSystemResourceUsage(t *testing.T) {
	pm := &mem.VirtualMemoryStat{Total: 10, Available: 9, Used: 1, UsedPercent: 10, Free: 8}
	d := &disk.UsageStat{Total: 20, Used: 5, UsedPercent: 25, Free: 15}
	cpuTimes := []cpu.TimesStat{{User: 1, System: 2, Idle: 3}}
	cpuPercent := []float64{4}
	ioCounters := []gnet.IOCountersStat{{BytesRecv: 5, BytesSent: 6}}

	got, err := newSystemResourceUsage(pm, cpuTimes, cpuPercent, d, ioCounters)
	if err != nil {
		t.Fatalf("newSystemResourceUsage returned error: %v", err)
	}
	if got.UsedCPUPercent != 4 || got.UserCPU != 1 || got.SystemCPU != 2 || got.IdleCPU != 3 {
		t.Fatalf("unexpected CPU stats: %+v", got)
	}
	if got.TotalRAM != 10 || got.TotalDisk != 20 || got.ReceivedBytesIO != 5 || got.WrittenBytesIO != 6 {
		t.Fatalf("unexpected resource stats: %+v", got)
	}
}
