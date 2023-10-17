package Executor

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type ProcessInfo struct {
	Command   string
	Argv      []string
	CmdDir    string
	StartTime time.Time
}
type Process struct {
	info     ProcessInfo
	cmd      *exec.Cmd
	out, err *bufio.Reader
	in       *io.WriteCloser
	ctx      context.Context
}

type Manager struct {
	pid       int
	processes *sync.Map
}

func (m *Manager) NewProcess(command string, args []string, cmdDir string) (int, error) {
	return m.NewProcessWithContext(context.Background(), command, args, cmdDir)
}

func (m *Manager) NewProcessWithContext(ctx context.Context, command string, args []string, cmdDir string) (int, error) {
	p := &Process{
		info: ProcessInfo{
			Command: command,
			Argv:    args,
			CmdDir:  cmdDir,
		},
		ctx: ctx,
	}
	cmd := exec.CommandContext(ctx, command, args...)

	cmd.Dir = cmdDir
	cmd.SysProcAttr = SysProcAttr
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	p.out = bufio.NewReader(outPipe)
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}
	p.err = bufio.NewReader(errPipe)

	writePipe, err := cmd.StdinPipe()
	if err != nil {
		return 0, err
	}
	p.in = &writePipe
	p.cmd = cmd
	err = p.cmd.Start()
	if err != nil {
		return 0, err
	}
	pid := m.pid
	m.pid += 1
	p.info.StartTime = time.Now()
	m.processes.Store(pid, p)
	return pid, nil
}

func (m *Manager) FetchStdout(pid int) ([]byte, error) {
	process, err := m.getProcess(pid)
	if err != nil {
		return nil, err
	}
	output, err := m.getOutput(process.ctx, process.out)
	if err != nil {
		return nil, err
	}
	return output, nil
}
func (m *Manager) FetchStderr(pid int) ([]byte, error) {
	process, err := m.getProcess(pid)
	if err != nil {
		return nil, err
	}
	output, err := m.getOutput(process.ctx, process.err)
	if err != nil {
		return nil, err
	}
	return output, nil
}
func (m *Manager) FetchAll(pid int) ([]byte, []byte, error) {
	stdout, err1 := m.FetchStdout(pid)
	stderr, err2 := m.FetchStderr(pid)
	if err1 != nil && err2 != nil {
		return nil, nil, err1
	} else {
		return stdout, stderr, nil
	}
}

func (m *Manager) WaitOutput(pid int) ([]byte, []byte, error) {
	var stdout, stderr []byte
	process, err := m.getProcess(pid)
	if err != nil {
		return nil, nil, err
	}
	for {
		output, err := m.getOutput(process.ctx, process.out)
		if err != nil {
			break
		}
		stdout = append(stdout, output...)
	}

	for {
		output, err := m.getOutput(process.ctx, process.err)
		if err != nil {
			break
		}
		stderr = append(stderr, output...)
	}

	return stdout, stderr, nil
}

func (m *Manager) getProcess(pid int) (*Process, error) {
	p, ok := m.processes.Load(pid)
	if !ok {
		return nil, errors.New("process '" + strconv.Itoa(pid) + "' not exists")
	}
	return p.(*Process), nil
}

// Running 检查是否正在运行
func (m *Manager) Running(pid int) bool {
	process, err := m.getProcess(pid)
	if err != nil {
		return false
	}
	if process.cmd.ProcessState != nil {
		if process.cmd.ProcessState.Exited() {
			//m.processes.Delete(pid)
			return false
		} else {
			return true
		}
	} else {
		return false
	}
}

func (m *Manager) Kill(pid int) error {
	process, err := m.getProcess(pid)
	if err != nil {
		return err
	}
	err = process.cmd.Process.Kill()
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) List() map[int]ProcessInfo {
	res := map[int]ProcessInfo{}
	m.processes.Range(func(key, value any) bool {
		res[key.(int)] = value.(*Process).info
		return true
	})
	return res
}

// getOutput todo:移动至Process
func (m *Manager) getOutput(ctx context.Context, reader *bufio.Reader) ([]byte, error) {
	outChan := make(chan []byte)
	errChan := make(chan error)
	go func() {
		outputBytes := make([]byte, 200)
		n, err := reader.Read(outputBytes)
		if err != nil {
			errChan <- err
		} else {
			outChan <- outputBytes[:n]
		}
	}()

	select {
	case output := <-outChan:
		return output, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, errors.New("timeout")
	}
}

func NewManager(name string) *Manager {
	return &Manager{
		pid:       1004,
		processes: &sync.Map{},
	}
}
