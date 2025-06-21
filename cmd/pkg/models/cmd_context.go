package models

import (
	"context"
)

type CommandContext struct {
	Context context.Context
	State   CommandState
}

type CommandState struct {
	LoggedIn      bool
	CurrRepoKnown bool
	User          *User
	Organization  *Organization
	Project       *Project
}
