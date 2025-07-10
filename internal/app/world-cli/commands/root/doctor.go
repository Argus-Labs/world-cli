package root

import (
	tea "github.com/charmbracelet/bubbletea"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/dependency"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/teacmd"
	"pkg.world.dev/world-cli/internal/pkg/tea/component/program"
	"pkg.world.dev/world-cli/internal/pkg/tea/style"
)

var DoctorDeps = []dependency.Dependency{
	dependency.Git,
	dependency.Go,
	dependency.Docker,
	dependency.DockerDaemon,
}

func (h *Handler) Doctor() error {
	p := program.NewTeaProgram(NewWorldDoctorModel())
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
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
