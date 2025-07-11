package root

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/editor"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/teacmd"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/tomlutil"
	"pkg.world.dev/world-cli/internal/pkg/tea/component/program"
	"pkg.world.dev/world-cli/internal/pkg/tea/component/steps"
	"pkg.world.dev/world-cli/internal/pkg/tea/style"
)

func (h *Handler) Create(directory string) error {
	p := program.NewTeaProgram(NewWorldCreateModel(directory))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

const TemplateGitURL = "https://github.com/Argus-Labs/starter-game-template.git"

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

//////////////////////
// Bubble Tea Model //
//////////////////////

type WorldCreateModel struct {
	logs             []string
	steps            steps.Model
	projectNameInput textinput.Model
	depStatus        []teacmd.DependencyStatus
	depStatusErr     error
}

func NewWorldCreateModel(directory string) WorldCreateModel {
	pnInput := textinput.New()
	pnInput.Prompt = style.DoubleRightIcon.Render()
	pnInput.Placeholder = "starter-game"
	pnInput.Focus()
	pnInput.Width = 50

	createSteps := steps.New()
	createSteps.Steps = []steps.Entry{
		steps.NewStep("Set game shard name"),
		steps.NewStep("Initialize game shard with starter-game-template"),
		steps.NewStep("Update world.toml configuration"),
		steps.NewStep("Set up Cardinal Editor"),
	}

	if directory != "" {
		// Extract just the directory name from the path
		dirName := filepath.Base(directory)
		pnInput.SetValue(dirName)
	}

	return WorldCreateModel{
		steps:            createSteps,
		projectNameInput: pnInput,
	}
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

// Init returns an initial command for the application to run.
func (m WorldCreateModel) Init() tea.Cmd {
	// If the project name was passed in as an argument, skip the 1st step
	if m.projectNameInput.Value() != "" {
		return tea.Sequence(textinput.Blink, m.steps.StartCmd(), m.steps.CompleteStepCmd(nil))
	}
	return tea.Sequence(textinput.Blink, m.steps.StartCmd())
}

// Update handles incoming events and updates the model accordingly.
//
//nolint:funlen,gocognit // Long function, but it's ok because it's structured
func (m WorldCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case teacmd.CheckDependenciesMsg:
		m.depStatus = msg.DepStatus
		m.depStatusErr = msg.Err
		if msg.Err != nil {
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type { //nolint:exhaustive // Missing are not relevant
		case tea.KeyEnter:
			if m.projectNameInput.Value() == "" {
				m.projectNameInput.SetValue("starter-game")
			}
			// Validate project name doesn't contain spaces
			if strings.Contains(m.projectNameInput.Value(), " ") {
				m.logs = append(m.logs, style.CrossIcon.Render()+"Project name cannot contain spaces")
				return m, tea.Quit
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
			err := teacmd.GitCloneCmd(TemplateGitURL, m.projectNameInput.Value(), "Initial commit from World CLI")
			teaCmd := func() tea.Msg {
				return teacmd.GitCloneFinishMsg{Err: err}
			}

			return m, tea.Sequence(
				NewLogCmd(style.ChevronIcon.Render()+"Cloning starter-game-template..."),
				teaCmd,
			)
		}
		if msg.Index == 2 {
			err := updateWorldToml(m.projectNameInput.Value())
			teaCmd := func() tea.Msg {
				return teacmd.GitCloneFinishMsg{Err: err}
			}

			return m, tea.Sequence(
				NewLogCmd(style.ChevronIcon.Render()+"Updating world.toml configuration..."),
				teaCmd,
			)
		}
		if msg.Index == 3 {
			err := editor.SetupCardinalEditor(".", "cardinal")
			teaCmd := func() tea.Msg {
				return teacmd.GitCloneFinishMsg{Err: err}
			}

			return m, tea.Sequence(
				NewLogCmd(style.ChevronIcon.Render()+"Setting up Cardinal Editor"),
				teaCmd,
			)
		}
		return m, nil

	case steps.SignalStepCompletedMsg:
		// If step 1 is completed, log success message
		if msg.Index == 1 {
			return m, NewLogCmd(style.ChevronIcon.Render() +
				"Successfully created a starter game shard in ./" + m.projectNameInput.Value())
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

// View renders the UI based on the data in the WorldCreateModel.
func (m WorldCreateModel) View() string {
	if m.depStatusErr != nil {
		return teacmd.PrettyPrintMissingDependency(m.depStatus)
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

func updateWorldToml(projectName string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Get absolute path to world.toml - it should be at the root of the cloned project
	absProjectDir := filepath.Join(cwd, "world.toml")

	// Update the forge section with the project name
	updates := map[string]interface{}{
		"PROJECT_NAME": projectName,
	}
	if err := tomlutil.UpdateTOMLSection(absProjectDir, "forge", updates); err != nil {
		return fmt.Errorf("failed to update world.toml: %w", err)
	}

	return nil
}
