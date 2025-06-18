package models

import (
	"context"

	"pkg.world.dev/world-cli/cmd/world/pkg/clients/config"
)

type CommandContext struct {
	Context context.Context
	State   CommandState
	Config  *config.Config
}

type CommandState struct {
	LoggedIn      bool
	CurrRepoKnown bool
	User          *User
	Organization  *Organization
	Project       *Project
}
