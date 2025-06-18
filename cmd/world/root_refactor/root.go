package root

import "github.com/alecthomas/kong"

var CLI Cmd

//nolint:lll // must be on one line
type Cmd struct {
	Create       *CreateCmd  `cmd:"" group:"Getting Started:"     help:"Create a new World Engine project"`
	Doctor       *DoctorCmd  `cmd:"" group:"Getting Started:"     help:"Check your development environment"`
	Login        *LoginCmd   `cmd:"" group:"Getting Started:"     help:"Login to World Forge, creating a new account if necessary"`
	kong.Plugins             // put this here so tools will be in the right place
	Version      *VersionCmd `cmd:"" group:"Additional Commands:" help:"Show the version of the CLI"`
	Verbose      bool        `                                    help:"Enable World CLI Debug logs"                               flag:"" short:"v"`
}

type CreateCmd struct {
	Parent    *Cmd   `kong:"-"`
	Directory string `         arg:"" optional:"" type:"path" help:"The directory to create the project in"`
}

func (c *CreateCmd) Run() error {
	// TODO: implement
	return nil
}

type DoctorCmd struct {
	Parent *Cmd `kong:"-"`
}

func (c *DoctorCmd) Run() error {
	// TODO: implement
	return nil
}

type VersionCmd struct {
	Parent *Cmd `kong:"-"`
	Check  bool `         help:"Check for the latest version of the CLI"`
}

func (c *VersionCmd) Run() error {
	// TODO: implement
	return nil
}

type LoginCmd struct {
	Parent *Cmd `kong:"-"`
}

func (c *LoginCmd) Run() error {
	// TODO: implement
	return nil
}
