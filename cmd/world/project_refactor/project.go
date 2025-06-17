package project

var CmdPlugin struct {
	Project *Cmd `cmd:"" aliases:"proj" group:"Project Commands:"      help:"Manage your projects"`
}

type Cmd struct {
	Create *CreateCmd `cmd:"" group:"Project Commands:" help:"Create a new project"`
	Switch *SwitchCmd `cmd:"" group:"Project Commands:" help:"Switch to a different project"`
	Update *UpdateCmd `cmd:"" group:"Project Commands:" help:"Update your project"`
	Delete *DeleteCmd `cmd:"" group:"Project Commands:" help:"Delete your project"`
}

type CreateCmd struct {
	Name      string `flag:"" help:"The name of the project"`
	Slug      string `flag:"" help:"The slug of the project"`
	AvatarURL string `flag:"" help:"The avatar URL of the project" type:"url"`
}

func (c *CreateCmd) Run() error {
	// TODO: implement
	return nil
}

type SwitchCmd struct {
	Slug string `flag:"" help:"The slug of the project to switch to"`
}

func (c *SwitchCmd) Run() error {
	// TODO: implement
	return nil
}

type UpdateCmd struct {
	Name      string `flag:"" help:"The new name of the project"`
	Slug      string `flag:"" help:"The new slug of the project"`
	AvatarURL string `flag:"" help:"The new avatar URL of the project" type:"url"`
}

func (c *UpdateCmd) Run() error {
	// TODO: implement
	return nil
}

type DeleteCmd struct {
}

func (c *DeleteCmd) Run() error {
	// TODO: implement
	return nil
}
