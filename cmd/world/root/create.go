package root

import (
	"pkg.world.dev/world-cli/internal/teacmd"
	"pkg.world.dev/world-cli/pkg/logger"
	"pkg.world.dev/world-cli/utils/tea/component/steps"
	"pkg.world.dev/world-cli/utils/tea/style"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
)

const TemplateGitUrl = "https://github.com/Argus-Labs/starter-game-template.git"

/////////////////
// Cobra Setup //
/////////////////

// createCmd creates a new World Engine project based on starter-game-template
// Usage: `world cardinal create [directory_name]`

func createCmd(teaCmd teacmd.TeaCmd) *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create [directory_name]",
		Short: "Create a World Engine game shard from scratch",
		Long: `Create a World Engine game shard based on https://github.com/Argus-Labs/starter-game-template.
If [directory_name] is set, it will automatically clone the starter project into that directory. 
Otherwise, it will prompt you to enter a directory name.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.SetDebugMode(cmd)
			p := tea.NewProgram(NewWorldCreateModel(teaCmd, args))
			if _, err := p.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	return createCmd
}

//////////////////////
// Bubble Tea Model //
//////////////////////

type WorldCreateModel struct {
	logs             []string
	steps            steps.Model
	projectNameInput textinput.Model
	args             []string
	depStatus        []teacmd.DependencyStatus
	depStatusErr     error
	err              error
	teaCmd           teacmd.TeaCmd
}

func NewWorldCreateModel(teaCmd teacmd.TeaCmd, args []string) WorldCreateModel {
	pnInput := textinput.New()
	pnInput.Prompt = style.DoubleRightIcon.Render()
	pnInput.Placeholder = "starter-game"
	pnInput.Focus()
	pnInput.Width = 50

	createSteps := steps.New()
	createSteps.Steps = []steps.Entry{
		steps.NewStep("Set game shard name"),
		steps.NewStep("Initialize game shard with starter-game-template"),
	}

	// Set the project text if it was passed in as an argument
	if len(args) == 1 {
		pnInput.SetValue(args[0])
	}

	return WorldCreateModel{
		steps:            createSteps,
		projectNameInput: pnInput,
		args:             args,
		teaCmd:           teaCmd,
	}
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

// Init returns an initial command for the application to run
func (m WorldCreateModel) Init() tea.Cmd {
	// If the project name was passed in as an argument, skip the 1st step
	if m.projectNameInput.Value() != "" {
		return tea.Sequence(textinput.Blink, m.steps.StartCmd(), m.steps.CompleteStepCmd(nil))
	}
	return tea.Sequence(textinput.Blink, m.steps.StartCmd())
}

// Update handles incoming events and updates the model accordingly
func (m WorldCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case teacmd.CheckDependenciesMsg:
		m.depStatus = msg.DepStatus
		m.depStatusErr = msg.Err
		if msg.Err != nil {
			return m, tea.Quit
		} else {
			return m, nil
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.projectNameInput.Value() == "" {
				m.projectNameInput.SetValue("starter-game")
			}
			m.projectNameInput.Blur()
			return m, m.steps.CompleteStepCmd(nil)
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case NewLogMsg:
		m.logs = append(m.logs, msg.Log)
		return m, nil

	case steps.SignalStepStartedMsg:
		// If step 1 is started, dispatch the git clone command
		if msg.Index == 1 {
			err := m.teaCmd.GitCloneCmd(TemplateGitUrl, m.projectNameInput.Value(), "Initial commit from World CLI")
			teaCmd := func() tea.Msg {
				return teacmd.GitCloneFinishMsg{Err: err}
			}

			return m, tea.Sequence(
				NewLogCmd(style.ChevronIcon.Render()+"Cloning starter-game-template..."),
				teaCmd,
			)
		}
		return m, nil

	case steps.SignalStepCompletedMsg:
		// If step 1 is completed, log success message
		if msg.Index == 1 {
			return m, NewLogCmd(style.ChevronIcon.Render() + "Successfully created a starter game shard in ./" + m.projectNameInput.Value())
		}

	case steps.SignalStepErrorMsg:
		// Log error, then quit
		return m, tea.Sequence(NewLogCmd(style.CrossIcon.Render()+"Error: "+msg.Err.Error()), tea.Quit)

	case steps.SignalAllStepCompletedMsg:
		// All done, quit
		return m, tea.Quit

	case teacmd.GitCloneFinishMsg:
		// If there is an error, log stderr then mark step as failed
		if msg.Err != nil {
			m.logs = append(m.logs, style.CrossIcon.Render()+msg.Err.Error())
			return m, m.steps.CompleteStepCmd(msg.Err)
		}

		// Otherwise, mark step as completed
		return m, m.steps.CompleteStepCmd(nil)

	default:
		var cmd tea.Cmd
		m.steps, cmd = m.steps.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.projectNameInput, cmd = m.projectNameInput.Update(msg)
	return m, cmd
}

// View renders the UI based on the data in the WorldCreateModel
func (m WorldCreateModel) View() string {
	if m.depStatusErr != nil {
		return m.teaCmd.PrettyPrintMissingDependency(m.depStatus)
	}

	output := ""
	output += m.steps.View()
	output += "\n\n"
	output += style.QuestionIcon.Render() + "What is your game shard name? " + m.projectNameInput.View()
	output += "\n\n"
	output += strings.Join(m.logs, "\n")
	output += "\n\n"

	return output
}

/////////////////////////
// Bubble Tea Commands //
/////////////////////////

type NewLogMsg struct {
	Log string
}

func NewLogCmd(log string) tea.Cmd {
	return func() tea.Msg {
		return NewLogMsg{Log: log}
	}
}
