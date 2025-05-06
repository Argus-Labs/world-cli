package program

import (
	"bytes"
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"golang.org/x/term"
	"pkg.world.dev/world-cli/common/printer"
)

// An interface describing the parts of BubbleTea's Program that we actually use.
type Program interface {
	Start() error
	Send(msg tea.Msg)
	Quit()
}

// A dumb text implementation of BubbleTea's Program that allows
// for output to be piped to another program.
type fakeProgram struct {
	model tea.Model
}

type (
	StatusMsg   string
	ProgressMsg *float64
	PsqlMsg     *string
)

type StatusWriter struct {
	Program
}

type logModel struct {
	cancel context.CancelFunc

	spinner     spinner.Model
	status      string
	progress    *progress.Model
	psqlOutputs []string

	width int
}

func NewProgram(model tea.Model, opts ...tea.ProgramOption) Program {
	var p Program
	if term.IsTerminal(int(os.Stdin.Fd())) {
		p = tea.NewProgram(model, opts...)
	} else {
		p = newFakeProgram(model)
	}
	return p
}

func newFakeProgram(model tea.Model) *fakeProgram {
	p := &fakeProgram{
		model: model,
	}
	return p
}

func (p *fakeProgram) Start() error {
	initCmd := p.model.Init()
	message := initCmd()
	if message != nil {
		p.model.Update(message)
	}
	return nil
}

func (p *fakeProgram) Send(msg tea.Msg) {
	switch msg := msg.(type) {
	case StatusMsg:
		printer.Infoln(string(msg))
	case PsqlMsg:
		if msg != nil {
			printer.Infoln(*msg)
		}
	}

	_, cmd := p.model.Update(msg)
	if cmd != nil {
		cmd()
	}
}

func (p *fakeProgram) Quit() {
	p.Send(tea.Quit())
}

func (t StatusWriter) Write(p []byte) (int, error) {
	trimmed := bytes.TrimRight(p, "\n")
	t.Send(StatusMsg(trimmed))
	return len(p), nil
}

func RunProgram(ctx context.Context, f func(p Program, ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	p := NewProgram(logModel{
		cancel: cancel,
		spinner: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
		),
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- f(p, ctx)
		p.Quit()
	}()

	if err := p.Start(); err != nil {
		return err
	}
	return <-errCh
}

func (m logModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m logModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type { //nolint:exhaustive // not applicable
		case tea.KeyCtrlC:
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		default:
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case spinner.TickMsg:
		spinnerModel, cmd := m.spinner.Update(msg)
		m.spinner = spinnerModel
		return m, cmd
	case progress.FrameMsg:
		if m.progress == nil {
			return m, nil
		}

		tmp, cmd := m.progress.Update(msg)
		progressModel, ok := tmp.(progress.Model)
		if ok {
			m.progress = &progressModel
		}
		return m, cmd
	case StatusMsg:
		m.status = string(msg)
		return m, nil
	case ProgressMsg:
		if msg == nil {
			m.progress = nil
			return m, nil
		}

		if m.progress == nil {
			progressModel := progress.New(progress.WithGradient("#1c1c1c", "#34b27b"))
			m.progress = &progressModel
		}

		return m, m.progress.SetPercent(*msg)
	case PsqlMsg:
		if msg == nil {
			m.psqlOutputs = []string{}
			return m, nil
		}

		m.psqlOutputs = append(m.psqlOutputs, *msg)
		// set max output length to 5
		if len(m.psqlOutputs) > 5 { //nolint:gomnd
			m.psqlOutputs = m.psqlOutputs[1:]
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m logModel) View() string {
	var progress string
	if m.progress != nil {
		progress = "\n\n" + m.progress.View()
	}

	var psqlOutputs string
	if len(m.psqlOutputs) > 0 {
		psqlOutputs = "\n\n" + strings.Join(m.psqlOutputs, "\n")
	}

	return wrap.String(m.spinner.View()+m.status+progress+psqlOutputs, m.width)
}
