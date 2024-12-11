package evm

import (
	"github.com/spf13/cobra"

	globalconfig "pkg.world.dev/world-cli/config"
	logger "pkg.world.dev/world-cli/logging"
	"pkg.world.dev/world-cli/ui/style"
)

const (
	FlagUseDevDA     = "dev"
	FlagDAAuthToken  = "da-auth-token" //nolint:gosec // false positive
	EnvDAAuthToken   = "DA_AUTH_TOKEN" //nolint:gosec // false positive
	EnvDABaseURL     = "DA_BASE_URL"
	EnvDANamespaceID = "DA_NAMESPACE_ID"
)

var BaseCmd = &cobra.Command{
	Use:     "evm",
	Short:   "Utilities for managing the EVM shard",
	Long:    style.CLIHeader("World CLI — EVM", "Utilities for managing the EVM shard"),
	GroupID: "core",
}

func init() {
	// Register subcommands - `world evm [subcommand]`
	BaseCmd.AddCommand(startCmd, stopCmd)
	registerConfigAndVerboseFlags(startCmd, stopCmd)
}

func registerConfigAndVerboseFlags(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		globalconfig.AddConfigFlag(cmd)
		logger.AddVerboseFlag(cmd)
	}
}
