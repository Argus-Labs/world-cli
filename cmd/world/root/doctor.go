package root

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/teacmd"
	"pkg.world.dev/world-cli/tea/style"
)

var DoctorDeps = []teacmd.Dependency{
	teacmd.GitDependency,
	teacmd.GoDependency,
	teacmd.DockerDependency,
	teacmd.DockerComposeDependency,
	teacmd.DockerDaemonDependency,
}

/////////////////
// Cobra Setup //
/////////////////

// doctorCmd checks that required dependencies are installed
// Usage: `world doctor`
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that required dependencies are installed",
	Long: `Check that required dependencies are installed.

World CLI requires the following dependencies to be installed:
- Git
- Go
- Docker`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(NewWorldDoctorModel())
		_, err := p.Run()
		if err != nil {
			return err
		}
		return nil
	},
}

//////////////////////
// Bubble Tea Model //
//////////////////////

type WorldDoctorModel struct {
	DepStatus    []teacmd.DependencyStatus
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
	return teacmd.CheckDependenciesCmd(DoctorDeps)
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
	depList, help := teacmd.PrintDependencyStatus(m.DepStatus)
	out := style.Container.Render("--- World CLI Doctor ---") + "\n\n"
	out += "Checking dependencies...\n"
	out += depList + "\n" + help + "\n"
	return out
}
