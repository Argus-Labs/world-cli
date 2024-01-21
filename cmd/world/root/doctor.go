package root

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/internal/teacmd"
	"pkg.world.dev/world-cli/pkg/logger"
	"pkg.world.dev/world-cli/utils/dependency"
	"pkg.world.dev/world-cli/utils/tea/style"
)

var DoctorDeps = []dependency.Dependency{
	&dependency.Git,
	&dependency.Go,
	&dependency.Docker,
	&dependency.DockerCompose,
	&dependency.DockerDaemon,
}

/////////////////
// Cobra Setup //
/////////////////

// doctorCmd checks that required dependencies are installed
// Usage: `world doctor`
func doctorCmd(teaCmd teacmd.TeaCmd) *cobra.Command {
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
			p := tea.NewProgram(NewWorldDoctorModel(teaCmd))
			_, err := p.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}

	logger.AddLogFlag(doctorCmd)

	return doctorCmd
}

//////////////////////
// Bubble Tea Model //
//////////////////////

type WorldDoctorModel struct {
	DepStatus    []teacmd.DependencyStatus
	DepStatusErr error
	teaCmd       teacmd.TeaCmd
}

func NewWorldDoctorModel(teaCmd teacmd.TeaCmd) WorldDoctorModel {
	return WorldDoctorModel{
		teaCmd: teaCmd,
	}
}

//////////////////////////
// Bubble Tea Lifecycle //
//////////////////////////

// Init returns an initial command for the application to run
func (m WorldDoctorModel) Init() tea.Cmd {
	return m.teaCmd.CheckDependenciesCmd(DoctorDeps)
}

// Update handles incoming events and updates the model accordingly
func (m WorldDoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case teacmd.CheckDependenciesMsg:
		m.DepStatus = msg.DepStatus
		m.DepStatusErr = msg.Err
		return m, tea.Quit
	}
	return m, nil
}

// View renders the model to the screen
func (m WorldDoctorModel) View() string {
	depList, help := m.teaCmd.PrintDependencyStatus(m.DepStatus)
	out := style.Container.Render("--- World CLI Doctor ---") + "\n\n"
	out += "Checking dependencies...\n"
	out += depList + "\n" + help + "\n"
	return out
}
