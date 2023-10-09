package utils

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guumaster/logsymbols"
)

type Status int32

const (
	PENDING Status = iota
	SUCCESS
	FAILED
)

type StatusObject struct {
	statusName string
	status     atomic.Int32
	check      func(*StatusObject)
}

func (s *StatusCollection) GetHeight() int {
	return s.width
}

func (s *StatusCollection) GetWidth() int {
	return s.height
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
	Style             lipgloss.Style
	Spinner           spinner.Model
	Statuses          []*StatusObject
	ShutdownOnChecked bool
	ShutdownChan      chan bool
	width             int
	height            int
}

func WithShutdownOnChecked(box StatusCollection) {
	box.ShutdownOnChecked = true
}

func NewStatusCollection(statuses []*StatusObject, options ...Option) *StatusCollection {
	res := StatusCollection{
		Style:             lipgloss.NewStyle().Align(lipgloss.Top, lipgloss.Left),
		Spinner:           spinner.New(spinner.WithSpinner(spinner.Pulse)),
		Statuses:          statuses,
		ShutdownChan:      make(chan bool),
		ShutdownOnChecked: false,
		width:             500,
		height:            500,
	}
	for _, option := range options {
		option(&res)
	}
	return &res
}

func (s *StatusCollection) IsAllChecked() bool {
	for _, status := range s.Statuses {
		if status.GetStatus() == PENDING {
			return false
		}
	}
	return true
}

func (s *StatusCollection) View() string {
	var acc string
	for _, status := range s.Statuses {
		acc += status.GetStatusMessage(&s.Spinner)
	}
	if s.IsAllChecked() {
		acc += "All dependencies found.\n"
	}
	return s.Style.Width(s.width).Height(s.height).Render(acc)
}

func (s *StatusCollection) Init() tea.Cmd {
	go func() {
	loop:
		for {
			time.Sleep(500 * time.Millisecond)
			for _, status := range s.Statuses {
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

func (s *StatusCollection) Shutdown() {
	s.ShutdownChan <- true
}

func (s *StatusCollection) SetStyle(style *lipgloss.Style) {
	s.Style = s.Style.Inherit(*style)
}

func (s *StatusCollection) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		var cmd tea.Cmd
		s.Spinner, cmd = s.Spinner.Update(msg)

		return s, cmd
	//case tea.KeyMsg:
	//	switch msg.String() {
	//	case "q", "ctrl+c":
	//		s.Shutdown()
	//		return s, tea.Quit
	//	}
	default:
		var cmd tea.Cmd
		s.Spinner, cmd = s.Spinner.Update(msg)
		if s.IsAllChecked() && s.ShutdownOnChecked {
			s.Shutdown()
			return s, tea.Quit
		}
		return s, cmd
	}
}
