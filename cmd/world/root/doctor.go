package root

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"io"
	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/tea_cmd"
	"pkg.world.dev/world-cli/tea/style"
)

var DoctorDeps = []dependency.Dependency{
	dependency.Git,
	dependency.Go,
	dependency.Docker,
	dependency.DockerCompose,
	dependency.DockerDaemon,
}

/////////////////
// Cobra Setup //
/////////////////

// doctorCmd checks that required dependencies are installed
// Usage: `world doctor`
func getDoctorCmd(writer io.Writer) *cobra.Command {
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check that required dependencies are installed",
		Long: `Check that required dependencies are installed.

World CLI requires the following dependencies to be installed:
- Git
- Go
- Docker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.SetDebugMode(cmd)
			p := tea.NewProgram(NewWorldDoctorModel(), tea.WithOutput(writer))
			_, err := p.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}

	return doctorCmd
}

//////////////////////
// Bubble Tea Model //
//////////////////////

type WorldDoctorModel struct {
	DepStatus    []tea_cmd.DependencyStatus
	DepStatusErr error
}

func NewWorldDoctorModel() WorldDoctorModel {
	return WorldDoctorModel{}
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

// Init returns an initial command for the application to run
func (m WorldDoctorModel) Init() tea.Cmd {
	return tea_cmd.CheckDependenciesCmd(DoctorDeps)
}

// Update handles incoming events and updates the model accordingly
func (m WorldDoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case tea_cmd.CheckDependenciesMsg:
		m.DepStatus = msg.DepStatus
		m.DepStatusErr = msg.Err
		return m, tea.Quit
	}
	return m, nil
}

// View renders the model to the screen
func (m WorldDoctorModel) View() string {
	depList, help := tea_cmd.PrintDependencyStatus(m.DepStatus)
	out := style.Container.Render("--- World CLI Doctor ---") + "\n\n"
	out += "Checking dependencies...\n"
	out += depList + "\n" + help + "\n"
	return out
}
