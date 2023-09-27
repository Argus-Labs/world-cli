package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/guumaster/logsymbols"
)

type Status int32

const (
	PENDING Status = iota
	SUCCESS
	FAILED
)

// RunShellCmd run a command using the shell; no need to split args
// from https://stackoverflow.com/questions/6182369/exec-a-shell-command-in-go
func RunShellCmd(cmd string, shell bool) []byte {
	logger := log.New(os.Stderr, "", 0)
	if shell {
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			switch v := err.(type) {
			case *exec.ExitError:
				logger.Fatalf("%s", string(v.Stderr))
			default:
				logger.Fatalf(err.Error())
			}
		}
		return out
	}
	out, err := exec.Command(cmd).Output()
	if err != nil {
		log.Fatal(err)
	}
	return out
}

type StatusObject struct {
	statusName string
	status     atomic.Int32
	check      func(*StatusObject)
}

func CreateNewStatus(statusName string, checkFunc func(*StatusObject)) *StatusObject {
	res := StatusObject{
		statusName: statusName,
		check:      checkFunc,
	}
	res.status.Store(int32(PENDING))
	return &res
}

func (s *StatusObject) AutoSetStatus() {
	if s.GetStatus() == PENDING {
		s.check(s)
	}
}

func (s *StatusObject) SetStatus(status Status) {
	s.status.Store(int32(status))
}

func (s *StatusObject) GetStatus() Status {
	return Status(s.status.Load())
}

func (s *StatusObject) GetStatusMessage(spinnerModel *spinner.Model) string {
	var prefix string
	switch s.GetStatus() {
	case PENDING:
		prefix = spinnerModel.View()
		break
	case SUCCESS:
		prefix = string(logsymbols.Success)
		break
	case FAILED:
		prefix = string(logsymbols.Error)
		break
	default:
		panic("logic error with GetStatusMessage, check enum utils.Status")
	}
	finalString := fmt.Sprintf("%s %s\n", prefix, s.statusName)
	return finalString
}

type StatusCollection struct {
	Spinner      spinner.Model
	Statuses     []*StatusObject
	ShutdownChan chan bool
}

func (c StatusCollection) IsAllChecked() bool {
	for _, status := range c.Statuses {
		if status.GetStatus() == PENDING {
			return false
		}
	}
	return true
}

func (s StatusCollection) View() string {
	var acc string
	for _, status := range s.Statuses {
		acc += status.GetStatusMessage(&s.Spinner)
	}
	return acc
}

func (s StatusCollection) Init() tea.Cmd {
	go func() {
	loop:
		for {
			time.Sleep(500 * time.Millisecond)
			for _, status := range s.Statuses {
				//time.Sleep(200 * time.Millisecond)
				status.AutoSetStatus()
			}
			select {
			case <-s.ShutdownChan:
				break loop
			default:
				continue
			}
		}
	}()
	return s.Spinner.Tick
}

func (s StatusCollection) Shutdown() {
	s.ShutdownChan <- true
}

func (s StatusCollection) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.QuitMsg:
		return s, tea.Quit
	default:
		var cmd tea.Cmd
		s.Spinner, cmd = s.Spinner.Update(msg)
		if s.IsAllChecked() {
			s.Shutdown()
			return s, tea.Quit
		}
		return s, cmd
	}
}
