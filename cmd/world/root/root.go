package root

import (
	"bytes"

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
	cmd *cobra.Command
}

type Root interface {
	Execute() error
	SetOut(out *bytes.Buffer)
	SetErr(out *bytes.Buffer)
	SetArgs(args []string)
}

func New() Root {
	logger.Println("Initializing Commands")

	// rootCmd represents the base command
	// Usage: `world`
	var rootCmd = &cobra.Command{
		Use:   "world",
		Short: "A swiss army knife for World Engine projects",
		Long:  style.CLIHeader("World CLI", "A swiss army knife for World Engine projects"),
	}

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

	return &root{
		rootCmd,
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func (r *root) Execute() error {
	if err := r.cmd.Execute(); err != nil {
		logger.Errors(err)
		return err
	}
	// print log stack
	logger.PrintLogs()

	return nil
}

func (r *root) SetOut(out *bytes.Buffer) {
	r.cmd.SetOut(out)
}

func (r *root) SetErr(out *bytes.Buffer) {
	r.cmd.SetErr(out)
}

func (r *root) SetArgs(args []string) {
	r.cmd.SetArgs(args)
}
