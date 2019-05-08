package pprof

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
#include <unistd.h>
#include <sys/types.h>
#include <pwd.h>
#include <stdlib.h>
*/
// import "C"

func init() {
}

var Proc *ProcInfo = NewProcInfo()

type ProcInfo struct {
	Base *ProcBase
	Cpu  *ProcCpu
	Mem  *ProcMem
}

func NewProcInfo() *ProcInfo {
	return &ProcInfo{}
}

func (this *ProcInfo) Init() {
	this.Base = &ProcBase{}
	this.Cpu = &ProcCpu{}
	this.Mem = &ProcMem{}
	this.Base.GetProcInfo()
}

type ProcMem struct {
	VmSize int //Virtual memory size
	VmRss  int //Resident set size
	VmData int //Size of data
	VmStk  int //Size of Stack
	VmExe  int //Size of text segments
	VmLib  int //Shared library code size
}

func (pm *ProcMem) Refresh() error {
	file, err := os.Open("/proc/" + Proc.Base.Pid + "/status")
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		if strings.Trim(fields[1], " ") == "0" {
			continue
		}
		switch strings.Trim(fields[0], ":") {
		case "VmSize":
			pm.VmSize, _ = strconv.Atoi(fields[1])
		case "VmRSS":
			pm.VmRss, _ = strconv.Atoi(fields[1])
		case "VmData":
			pm.VmData, _ = strconv.Atoi(fields[1])
		case "VmStk":
			pm.VmStk, _ = strconv.Atoi(fields[1])
		case "VmExe":
			pm.VmExe, _ = strconv.Atoi(fields[1])
		case "VmLib":
			pm.VmLib, _ = strconv.Atoi(fields[1])
		}
	}
	return nil
}

func (pm *ProcMem) ReSet() {
	pm.VmData = 0
	pm.VmSize = 0
	pm.VmRss = 0
	pm.VmLib = 0
	pm.VmExe = 0
	pm.VmStk = 0
}

func (pm *ProcMem) String() string {
	return fmt.Sprintf("VIRT:%d KB, RES:%d KB, Data:%d KB, Stack:%d KB, Text Segment:%d KB, Lib:%d KB",
		pm.VmSize, pm.VmRss, pm.VmData, pm.VmStk, pm.VmExe, pm.VmLib)
}

type ProcCpu struct {
	Utime     uint64
	Stime     uint64
	Cutime    uint64
	Cstime    uint64
	StartTime uint64
	LastUS    uint64    //Utime+Stime
	LastTimer time.Time //time.Now()
}

func (pc *ProcCpu) Refresh() error {
	file, err := os.Open("/proc/" + Proc.Base.Pid + "/stat")
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil
	}
	fields := strings.Fields(line)
	if utime, err := strconv.ParseUint(fields[13], 10, 64); err == nil {
		pc.Utime = utime
	}
	if stime, err := strconv.ParseUint(fields[14], 10, 64); err == nil {
		pc.Stime = stime
	}
	if cutime, err := strconv.ParseUint(fields[15], 10, 64); err == nil {
		pc.Cutime = cutime
	}
	if cstime, err := strconv.ParseUint(fields[16], 10, 64); err == nil {
		pc.Cstime = cstime
	}
	if starttime, err := strconv.ParseUint(fields[21], 10, 64); err == nil {
		pc.StartTime = starttime
	}
	return nil
}

func (pc *ProcCpu) ReSet() {
	pc.Utime = 0
	pc.Stime = 0
	pc.Cutime = 0
	pc.Cstime = 0
	pc.StartTime = 0
}

/**
 * 采样时间段内的cpu使用率,算法与top命令一致。
 * top计算进程cpu使用率源码参见:procs的top.c:prochlp()
 */
func (pc *ProcCpu) CurrentUsage() float64 {
	nowTime := time.Now()
	totalTime := pc.Utime + pc.Stime
	sub := nowTime.Sub(pc.LastTimer).Seconds()
	sec := 100 / (float64(Machine.Hertz) * sub)
	pcpu := float64(totalTime) - float64(pc.LastUS)
	pc.LastUS = totalTime
	pc.LastTimer = nowTime
	return pcpu * sec
}

func (pc *ProcCpu) String() string {
	return fmt.Sprintf("Cpu:%0.2f%%", pc.CurrentUsage())
}

type ProcBase struct {
	Pid     string
	PPid    string
	Command string
	State   string
}

func (pb *ProcBase) GetProcInfo() error {
	pb.Pid = strconv.Itoa(os.Getpid())
	file, err := os.Open("/proc/" + pb.Pid + "/stat")
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil
	}
	fields := strings.Fields(line)
	pb.PPid = fields[3]
	pb.Command = pb.GetCommand()
	pb.State = fields[2]
	return nil
}

func (pb *ProcBase) GetCommand() string {
	command, _ := ioutil.ReadFile("/proc/" + pb.Pid + "/cmdline")
	return string(command)
}

type MachineCpu struct {
	User        uint64
	Nice        uint64
	System      uint64
	Idle        uint64
	Iowait      uint64
	Irq         uint64
	SoftIrq     uint64
	Stealstolen uint64
	Guest       uint64
}

func (mc *MachineCpu) Refresh() error {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil
	}
	fields := strings.Fields(line)

	if user, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
		mc.User = user
	}
	if nice, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
		mc.Nice = nice
	}
	if system, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
		mc.System = system
	}
	if idle, err := strconv.ParseUint(fields[4], 10, 64); err == nil {
		mc.Idle = idle
	}
	if iowait, err := strconv.ParseUint(fields[5], 10, 64); err == nil {
		mc.Iowait = iowait
	}
	if irq, err := strconv.ParseUint(fields[6], 10, 64); err == nil {
		mc.Irq = irq
	}
	if softirq, err := strconv.ParseUint(fields[7], 10, 64); err == nil {
		mc.SoftIrq = softirq
	}
	if stealstolen, err := strconv.ParseUint(fields[8], 10, 64); err == nil {
		mc.Stealstolen = stealstolen
	}
	if guest, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
		mc.Guest = guest
	}
	return nil
}

func (mc *MachineCpu) ReSet() {
	mc.User = 0
	mc.Nice = 0
	mc.System = 0
	mc.Idle = 0
	mc.Iowait = 0
	mc.Irq = 0
	mc.SoftIrq = 0
	mc.Stealstolen = 0
	mc.Guest = 0
}

var Machine *MachineInfo = NewMachineInfo()

type MachineInfo struct {
	Uptime float64
	Hertz  int
	Cpu    *MachineCpu
}

func NewMachineInfo() *MachineInfo {
	// return &MachineInfo{Hertz: int(C.sysconf(C._SC_CLK_TCK)), Cpu: &MachineCpu{}}
	return &MachineInfo{Hertz: 100, Cpu: &MachineCpu{}}
}

func (this *MachineInfo) GetUptime() float64 {
	if uptime, err := ioutil.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(uptime))
		this.Uptime, _ = strconv.ParseFloat(fields[0], 64)
	}
	return this.Uptime
}
