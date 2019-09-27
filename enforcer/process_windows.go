// +build windows

package enforcer

import (
	"context"
	"fmt"

	"github.com/gentlemanautomaton/winproc"
)

// PID is a process ID.
type PID = winproc.ID

// UniqueID is a unique process ID.
type UniqueID = winproc.UniqueID

// Ref is a process reference.
type Ref = winproc.Ref

// ProcessData holds information about a windows process.
type ProcessData = winproc.Process

// Process manages enforcement a process for which policies are being enforced.
type Process struct {
	data   ProcessData
	logger Logger
	ref    *winproc.Ref
}

// NewProcess returns a new process.
func NewProcess(data ProcessData, passive bool, logger Logger) (*Process, error) {
	// Open a reference to the process with the highest level of privilege
	// that we can get
	ref, err := openProcess(data.ID, passive)
	if err != nil {
		return nil, err
	}

	// Obtain a unique ID for the process.
	uid, err := ref.UniqueID()
	if err != nil {
		ref.Close()
		return nil, fmt.Errorf("unable to retrieve unique ID for process: %v", err)
	}

	// Verify that the unique ID matches our expectation
	if data.UniqueID() != uid {
		// The process ID was recycled into a new process. Abort.
		ref.Close()
		return nil, fmt.Errorf("the process to be managed has terminated")
	}

	return &Process{
		data:   data,
		ref:    ref,
		logger: logger,
	}, nil
}

// Data returns information about the process.
func (p *Process) Data() ProcessData {
	return p.data
}

// Wait waits until the context is cancelled or the process has terminated.
func (p *Process) Wait(ctx context.Context) error {
	return p.ref.Wait(ctx)
}

// Close releases any resources consumed by the process without killing it.
func (p *Process) Close() error {
	return p.ref.Close()
}

// Terminate causes the process to exit.
func (p *Process) Terminate() error {
	return p.ref.Terminate(5877)
}

func (p *Process) log(format string, v ...interface{}) {
	if p.logger == nil {
		return
	}
	p.logger.Log(ProcessEvent{
		ProcessID:   p.data.ID,
		ProcessName: p.data.Name,
		Msg:         fmt.Sprintf(format, v...),
	})
}

func (p *Process) debug(format string, v ...interface{}) {
	if p.logger == nil {
		return
	}
	p.logger.Log(ProcessEvent{
		ProcessID:   p.data.ID,
		ProcessName: p.data.Name,
		Msg:         fmt.Sprintf(format, v...),
		Debug:       true,
	})
}
