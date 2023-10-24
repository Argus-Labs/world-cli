package cmd

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"io"
	"pkg.world.dev/world-cli/cmd/action"
	"pkg.world.dev/world-cli/cmd/component/steps"
	"pkg.world.dev/world-cli/cmd/style"
	"strings"
)

const TemplateGitUrl = "https://github.com/Argus-Labs/starter-game-template.git"

/////////////////
// Cobra Setup //
/////////////////

func init() {
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a newModel game shard from scratch.",
	Long:  `Creates a World Engine game shard based on https://github.com/Argus-Labs/starter-game-template`,
	RunE: func(_ *cobra.Command, args []string) error {
		p := tea.NewProgram(NewWorldCreateModel(args))
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

//////////////////////
// Bubble Tea Model //
//////////////////////

type WorldCreateModel struct {
	logs             []string
	steps            steps.Model
	projectNameInput textinput.Model
	args             []string
	err              error
}

func NewWorldCreateModel(args []string) WorldCreateModel {
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
		err:              nil,
	}
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

// Init returns an initial command for the application to run
func (m WorldCreateModel) Init() tea.Cmd {
	// If the project name was passed in as an argument, skip the 1st step
	if m.projectNameInput.Value() != "" {
		return tea.Batch(textinput.Blink, m.steps.StartCmd(), m.steps.CompleteStepCmd(nil))
	}

	return tea.Batch(textinput.Blink, m.steps.StartCmd())
}

// Update handles incoming events and updates the model accordingly
func (m WorldCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
			return m, tea.Sequence(
				NewLogCmd(style.ChevronIcon.Render()+"Cloning starter-game-template..."),
				action.GitCloneCmd(TemplateGitUrl, m.projectNameInput.Value(), "Initial commit from World CLI"),
			)
		}
		return m, nil
	case steps.SignalStepCompletedMsg:
		// If step 1 is completed, log success message
		if msg.Index == 1 {
			return m, NewLogCmd(style.ChevronIcon.Render() + "Successfully created a newModel game shard based on starter-game-template!")
		}
	case steps.SignalStepErrorMsg:
		// Log error, then quit
		return m, tea.Sequence(NewLogCmd(style.CrossIcon.Render()+"Error: "+msg.Err.Error()), tea.Quit)
	case steps.SignalAllStepCompletedMsg:
		// All done, quit
		return m, tea.Quit
	case action.GitCloneFinishMsg:
		// If there is an error, log stderr then mark step as failed
		if msg.Err != nil {
			stderrBytes, err := io.ReadAll(msg.ErrBuf)
			if err != nil {
				m.logs = append(m.logs, style.CrossIcon.Render()+"Error occurred while reading stderr")
			} else {
				m.logs = append(m.logs, style.CrossIcon.Render()+string(stderrBytes))
			}
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
