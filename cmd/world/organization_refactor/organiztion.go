package organization

var CmdPlugin struct {
	Organization *Cmd `cmd:"" aliases:"org"  group:"Organization Commands:" help:"Manage your organizations"`
}

type Cmd struct {
	Create *CreateCmd `cmd:"" group:"Organization Commands:" help:"Create a new organization"`
	Switch *SwitchCmd `cmd:"" group:"Organization Commands:" help:"Switch to an organization"`
}

type CreateCmd struct {
	Name      string `flag:"" help:"The name of the organization"`
	Slug      string `flag:"" help:"The slug of the organization"`
	AvatarURL string `flag:"" help:"The avatar URL of the organization" type:"url"`
}

func (c *CreateCmd) Run() error {
	// TODO: implement
	return nil
}

type SwitchCmd struct {
	Slug string `flag:"" help:"The slug of the organization to switch to"`
}

func (c *SwitchCmd) Run() error {
	// TODO: implement
	return nil
}
