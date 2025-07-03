package evm

import (
	"context"

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

//nolint:revive // this is the "evm" command within the "world" command, properly named
type EvmCmd struct {
	Config string `flag:"" help:"A TOML config file"`

	Start *StartCmd `cmd:"" group:"EVM Commands:" help:"Launch your EVM blockchain environment"`
	Stop  *StopCmd  `cmd:"" group:"EVM Commands:" help:"Shut down your EVM blockchain environment"`
}

//nolint:lll // needed to put all the help text in the same line
type StartCmd struct {
	Parent      *EvmCmd         `kong:"-"`
	DAAuthToken string          `         flag:"" optional:"" help:"The DA Auth Token that allows the rollup to communicate with the Celestia client."`
	UseDevDA    bool            `         flag:"" optional:"" help:"Use a locally running DA layer"                                                    name:"dev"`
	Context     context.Context `kong:"-"`
}

func (c *StartCmd) Run() error {
	return Start(c)
}

type StopCmd struct {
	Parent *EvmCmd `kong:"-"`
}

func (c *StopCmd) Run() error {
	return Stop(c)
}
