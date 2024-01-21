package root

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/cmd/world/cardinal"
	"pkg.world.dev/world-cli/cmd/world/evm"
	"pkg.world.dev/world-cli/config"
	"pkg.world.dev/world-cli/internal/teacmd"
	"pkg.world.dev/world-cli/pkg/logger"
	"pkg.world.dev/world-cli/utils/tea/style"
	"pkg.world.dev/world-cli/utils/terminal"
)

type root struct {
}

type Root interface {
	Execute()
}

func New() Root {
	logger.Println("Initializing Commands")

	// Enable case-insensitive commands
	cobra.EnableCaseInsensitive = true

	// Register groups
	rootCmd.AddGroup(&cobra.Group{ID: "Core", Title: "World CLI Commands:"})

	//Initialize Dependencies
	terminalUtil := terminal.New()
	teaCmd := teacmd.New(terminalUtil)

	//Initialize Commands
	cardinalCmd := cardinal.New(terminalUtil, teaCmd)
	evmCmd := evm.New(terminalUtil, teaCmd)

	// Register base commands
	doctor := doctorCmd(teaCmd)
	create := createCmd(teaCmd)
	rootCmd.AddCommand(create, doctor, versionCmd)

	// Register subcommands
	rootCmd.AddCommand(cardinalCmd.GetBaseCmd())
	rootCmd.AddCommand(evmCmd.GetBaseCmd())

	config.AddConfigFlag(rootCmd)

	// Add --debug flag
	logger.AddLogFlag(create)
	logger.AddLogFlag(doctor)

	return &root{}
}

// rootCmd represents the base command
// Usage: `world`
var rootCmd = &cobra.Command{
	Use:   "world",
	Short: "A swiss army knife for World Engine projects",
	Long:  style.CLIHeader("World CLI", "A swiss army knife for World Engine projects"),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func (r *root) Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Errors(err)
	}
	// print log stack
	logger.PrintLogs()
}
