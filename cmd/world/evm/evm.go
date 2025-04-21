package evm

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/teacmd"
	"pkg.world.dev/world-cli/tea/style"
)

const (
	FlagUseDevDA     = "dev"
	FlagDAAuthToken  = "da-auth-token" //nolint:gosec // false positive
	EnvDAAuthToken   = "DA_AUTH_TOKEN" //nolint:gosec // false positive
	EnvDABaseURL     = "DA_BASE_URL"
	EnvDANamespaceID = "DA_NAMESPACE_ID"

	daService = teacmd.DockerServiceDA
)

var BaseCmd = &cobra.Command{
	Use:     "evm",
	Short:   "Streamlined tools for EVM blockchain integration",
	Long:    style.CLIHeader("World CLI — EVM", "Manage your EVM blockchain integration with ease"),
	GroupID: "core",
}

func init() {
	// Register subcommands - `world evm [subcommand]`
	BaseCmd.AddCommand(startCmd, stopCmd)
	registerConfigAndVerboseFlags(startCmd, stopCmd)
}

func registerConfigAndVerboseFlags(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		config.AddConfigFlag(cmd)
		logger.AddVerboseFlag(cmd)
	}
}
