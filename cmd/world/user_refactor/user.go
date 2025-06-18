package user

var CmdPlugin struct {
	User *Cmd `cmd:""`
}

//nolint:lll // needed to put all the help text in the same line
type Cmd struct {
	Invite *InviteToOrganizationCmd     `cmd:"" group:"User Commands:" optional:"" help:"Invite a user to an organization"`
	Role   *ChangeRoleInOrganizationCmd `cmd:"" group:"User Commands:" optional:"" help:"Change a user's role in an organization"`
	Update *UpdateCmd                   `cmd:"" group:"User Commands:" optional:"" help:"Update a user"`
}

type InviteToOrganizationCmd struct {
	Email string `flag:"" help:"The email of the user to invite"`
	Role  string `flag:"" help:"The role of the user to invite"`
}

func (c *InviteToOrganizationCmd) Run() error {
	// TODO: implement
	return nil
}

type ChangeRoleInOrganizationCmd struct {
	Email string `flag:"" help:"The email of the user to change the role of"`
	Role  string `flag:"" help:"The new role of the user"`
}

func (c *ChangeRoleInOrganizationCmd) Run() error {
	// TODO: implement
	return nil
}

type UpdateCmd struct {
	Name      string `flag:"" help:"The new name of the user"`
	AvatarURL string `flag:"" help:"The new avatar URL of the user" type:"url"`
}

func (c *UpdateCmd) Run() error {
	// TODO: implement
	return nil
}
