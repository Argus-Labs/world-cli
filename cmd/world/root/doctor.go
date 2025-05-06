package root

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/cmd/world/forge"
	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/common/teacmd"
	"pkg.world.dev/world-cli/tea/style"
)

var DoctorDeps = []dependency.Dependency{
	dependency.Git,
	dependency.Go,
	dependency.Docker,
	dependency.DockerDaemon,
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

// Init returns an initial command for the application to run.
func (m WorldDoctorModel) Init() tea.Cmd {
	return teacmd.CheckDependenciesCmd(DoctorDeps)
}

// Update handles incoming events and updates the model accordingly.
func (m WorldDoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type { //nolint:gocritic,exhaustive // cleaner with switch
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

// View renders the model to the screen.
func (m WorldDoctorModel) View() string {
	depList, help := teacmd.PrintDependencyStatus(m.DepStatus)
	out := style.Container.Render("--- World CLI Doctor ---") + "\n\n"
	out += "Checking dependencies...\n"
	out += depList + "\n" + help + "\n"
	return out
}

/////////////////
// Cobra Setup //
/////////////////

// Usage: `world doctor`.
func getDoctorCmd(writer io.Writer) *cobra.Command {
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Verify your development environment is ready",
		Long: `Diagnose and verify that your system has all required dependencies installed.

This command performs a comprehensive check of your development environment to ensure
you have everything needed to use World CLI effectively. It verifies the presence and
proper configuration of:

- Git: For version control and project management
- Go: Required for building and running World Engine projects
- Docker: Used for containerizing and running your game services

If any dependencies are missing, you'll receive guidance on how to install them.`,
		GroupID: "starter",
		RunE: func(_ *cobra.Command, _ []string) error {
			p := forge.NewTeaProgram(NewWorldDoctorModel(), tea.WithOutput(writer))
			_, err := p.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}

	return doctorCmd
}

func GetDoctorCmdTesting(writer io.Writer) *cobra.Command {
	return getDoctorCmd(writer)
}
