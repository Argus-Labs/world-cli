package evm

import (
	"pkg.world.dev/world-cli/common/teacmd"
)

const (
	FlagUseDevDA     = "dev"
	FlagDAAuthToken  = "da-auth-token" //nolint:gosec // false positive
	EnvDAAuthToken   = "DA_AUTH_TOKEN" //nolint:gosec // false positive
	EnvDABaseURL     = "DA_BASE_URL"
	EnvDANamespaceID = "DA_NAMESPACE_ID"

	daService = teacmd.DockerServiceDA
)

var EvmCmdPlugin struct {
	Evm *EvmCmd `cmd:"" group:"EVM Commands:" help:"Manage your EVM blockchain environment"`
}

type EvmCmd struct {
	Config string `flag:"" help:"A TOML config file"`

	Start *StartCmd `cmd:"" group:"Management Commands:" help:"Launch your EVM blockchain environment"`
	Stop  *StopCmd  `cmd:"" group:"Management Commands:" help:"Shut down your EVM blockchain environment"`
}
