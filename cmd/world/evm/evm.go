package evm

import (
	"github.com/spf13/cobra"

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
	Short:   "Utilities for managing the EVM shard",
	Long:    style.CLIHeader("World CLI — EVM", "Utilities for managing the EVM shard"),
	GroupID: "core",
}

func init() {
	// Register subcommands - `world evm [subcommand]`
	BaseCmd.AddCommand(startCmd, stopCmd)
}
