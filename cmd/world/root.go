package main

import (
	"context"

	"github.com/alecthomas/kong"
	cmdsetup "pkg.world.dev/world-cli/internal/app/world-cli/controllers/cmd_setup"
)

var CLI RootCmd

//nolint:lll // must be on one line
type RootCmd struct {
	Create       *CreateWorldCmd `cmd:"" group:"Getting Started:"     help:"Create a new World Engine project"`
	Doctor       *DoctorCmd      `cmd:"" group:"Getting Started:"     help:"Check your development environment"`
	Login        *LoginCmd       `cmd:"" group:"Getting Started:"     help:"Login to World Forge, creating a new account if necessary"`
	kong.Plugins                 // put this here so tools will be in the right place
	Version      *VersionCmd     `cmd:"" group:"Additional Commands:" help:"Show the version of the CLI"`
	Verbose      bool            `                                    help:"Enable World CLI Debug logs"                               flag:"" short:"v"`
}

//nolint:lll // must be on one line
type CreateWorldCmd struct {
	Parent       *RootCmd              `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Directory    string                `         arg:"" optional:"" type:"path" help:"The directory to create the project in"`
}

func (c *CreateWorldCmd) Run() error {
	return c.Dependencies.RootHandler.Create(c.Directory)
}

type DoctorCmd struct {
	Parent       *RootCmd              `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *DoctorCmd) Run() error {
	return c.Dependencies.RootHandler.Doctor()
}

type VersionCmd struct {
	Parent       *RootCmd              `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Check        bool                  `         help:"Check for the latest version of the CLI"`
}

func (c *VersionCmd) Run() error {
	return c.Dependencies.RootHandler.Version(c.Check)
}

type LoginCmd struct {
	Parent       *RootCmd              `kong:"-"`
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *LoginCmd) Run() error {
	return c.Dependencies.RootHandler.Login(c.Context)
}
