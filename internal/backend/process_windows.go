package backend

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

const createNoWindow = 0x08000000

func configure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP}
}

// Windows has no process groups to signal; its answer is the job object. Each server run gets one, with the agents
// and terminals it starts (children join their parent's job), and the job is set to kill everything in it when its
// last handle closes — which the kernel does for us when the app exits or crashes, so nothing the app started is
// orphaned however the app ends. A process that deliberately leaves with CREATE_BREAKAWAY_FROM_JOB (an
// intentionally detached tool) is allowed to: BREAKAWAY_OK.
const (
	jobObjectExtendedLimitInformation = 9
	jobKillOnClose                    = 0x00002000
	jobBreakawayOK                    = 0x00000800
	processSetQuota                   = 0x0100
	processTerminate                  = 0x0001
)

type basicLimits struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type extendedLimits struct {
	Basic                 basicLimits
	IoInfo                [6]uint64
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObject   = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJob = kernel32.NewProc("SetInformationJobObject")
	procAssignProcess     = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJob      = kernel32.NewProc("TerminateJobObject")

	jobs sync.Map // *exec.Cmd → syscall.Handle
)

// adopt puts a just-started server in a job of its own and returns the function that ends the job's life (closing
// the handle kills whatever is left in it). When any step fails the server runs unconfined, as before, and the
// error says so: the app must still work, and signalGroup falls back to killing the process alone.
func adopt(cmd *exec.Cmd) (release func(), err error) {
	release = func() {}
	if cmd.Process == nil {
		return release, fmt.Errorf("no process")
	}
	job, _, e := procCreateJobObject.Call(0, 0)
	if job == 0 {
		return release, fmt.Errorf("CreateJobObject: %w", e)
	}
	h := syscall.Handle(job)
	var lim extendedLimits
	lim.Basic.LimitFlags = jobKillOnClose | jobBreakawayOK
	if r, _, e := procSetInformationJob.Call(job, jobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&lim)), unsafe.Sizeof(lim)); r == 0 {
		_ = syscall.CloseHandle(h)
		return release, fmt.Errorf("SetInformationJobObject: %w", e)
	}
	ph, e2 := syscall.OpenProcess(processSetQuota|processTerminate, false, uint32(cmd.Process.Pid))
	if e2 != nil {
		_ = syscall.CloseHandle(h)
		return release, fmt.Errorf("OpenProcess: %w", e2)
	}
	defer syscall.CloseHandle(ph)
	if r, _, e := procAssignProcess.Call(job, uintptr(ph)); r == 0 {
		_ = syscall.CloseHandle(h)
		return release, fmt.Errorf("AssignProcessToJobObject: %w", e)
	}
	jobs.Store(cmd, h)
	return func() {
		if _, ok := jobs.LoadAndDelete(cmd); ok {
			_ = syscall.CloseHandle(h) // KILL_ON_JOB_CLOSE ends anything the server left behind
		}
	}, nil
}

// signalGroup ends the server and everything in its job; with no job, the server alone. There is no gentler
// signal for a windowless process, so a graceful stop is the same as a kill.
func signalGroup(cmd *exec.Cmd, kill bool) {
	if h, ok := jobs.Load(cmd); ok {
		_, _, _ = procTerminateJob.Call(uintptr(h.(syscall.Handle)), 1)
		return
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
