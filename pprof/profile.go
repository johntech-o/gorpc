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

func (this *ProcMem) Refresh() error {
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
			this.VmSize, _ = strconv.Atoi(fields[1])
		case "VmRSS":
			this.VmRss, _ = strconv.Atoi(fields[1])
		case "VmData":
			this.VmData, _ = strconv.Atoi(fields[1])
		case "VmStk":
			this.VmStk, _ = strconv.Atoi(fields[1])
		case "VmExe":
			this.VmExe, _ = strconv.Atoi(fields[1])
		case "VmLib":
			this.VmLib, _ = strconv.Atoi(fields[1])
		}
	}
	return nil
}

func (this *ProcMem) ReSet() {
	this.VmData = 0
	this.VmSize = 0
	this.VmRss = 0
	this.VmLib = 0
	this.VmExe = 0
	this.VmStk = 0
}

func (this *ProcMem) String() string {
	return fmt.Sprintf("VIRT:%d KB, RES:%d KB, Data:%d KB, Stack:%d KB, Text Segment:%d KB, Lib:%d KB",
		this.VmSize, this.VmRss, this.VmData, this.VmStk, this.VmExe, this.VmLib)
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

func (this *ProcCpu) Refresh() error {
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
		this.Utime = utime
	}
	if stime, err := strconv.ParseUint(fields[14], 10, 64); err == nil {
		this.Stime = stime
	}
	if cutime, err := strconv.ParseUint(fields[15], 10, 64); err == nil {
		this.Cutime = cutime
	}
	if cstime, err := strconv.ParseUint(fields[16], 10, 64); err == nil {
		this.Cstime = cstime
	}
	if starttime, err := strconv.ParseUint(fields[21], 10, 64); err == nil {
		this.StartTime = starttime
	}
	return nil
}

func (this *ProcCpu) ReSet() {
	this.Utime = 0
	this.Stime = 0
	this.Cutime = 0
	this.Cstime = 0
	this.StartTime = 0
}

/**
 * 采样时间段内的cpu使用率,算法与top命令一致。
 * top计算进程cpu使用率源码参见:procs的top.c:prochlp()
 */
func (this *ProcCpu) CurrentUsage() float64 {
	nowTime := time.Now()
	totalTime := this.Utime + this.Stime
	sub := nowTime.Sub(this.LastTimer).Seconds()
	sec := 100 / (float64(Machine.Hertz) * sub)
	pcpu := float64(totalTime) - float64(this.LastUS)
	this.LastUS = totalTime
	this.LastTimer = nowTime
	return pcpu * sec
}

func (this *ProcCpu) String() string {
	return fmt.Sprintf("Cpu:%0.2f%%", this.CurrentUsage())
}

type ProcBase struct {
	Pid     string
	PPid    string
	Command string
	State   string
}

func (this *ProcBase) GetProcInfo() error {
	this.Pid = strconv.Itoa(os.Getpid())
	file, err := os.Open("/proc/" + this.Pid + "/stat")
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
	this.PPid = fields[3]
	this.Command = this.GetCommand()
	this.State = fields[2]
	return nil
}

func (this *ProcBase) GetCommand() string {
	command, _ := ioutil.ReadFile("/proc/" + this.Pid + "/cmdline")
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

func (this *MachineCpu) Refresh() error {
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
		this.User = user
	}
	if nice, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
		this.Nice = nice
	}
	if system, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
		this.System = system
	}
	if idle, err := strconv.ParseUint(fields[4], 10, 64); err == nil {
		this.Idle = idle
	}
	if iowait, err := strconv.ParseUint(fields[5], 10, 64); err == nil {
		this.Iowait = iowait
	}
	if irq, err := strconv.ParseUint(fields[6], 10, 64); err == nil {
		this.Irq = irq
	}
	if softirq, err := strconv.ParseUint(fields[7], 10, 64); err == nil {
		this.SoftIrq = softirq
	}
	if stealstolen, err := strconv.ParseUint(fields[8], 10, 64); err == nil {
		this.Stealstolen = stealstolen
	}
	if guest, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
		this.Guest = guest
	}
	return nil
}

func (this *MachineCpu) ReSet() {
	this.User = 0
	this.Nice = 0
	this.System = 0
	this.Idle = 0
	this.Iowait = 0
	this.Irq = 0
	this.SoftIrq = 0
	this.Stealstolen = 0
	this.Guest = 0
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
