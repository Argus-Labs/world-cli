package user

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/cmd/world/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

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
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Email        string                `         flag:"" help:"The email of the user to invite"`
	Role         string                `         flag:"" help:"The role of the user to invite"`
}

func (c *InviteToOrganizationCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.Ignore,
	}

	flags := models.InviteUserToOrganizationFlags{
		Email: c.Email,
		Role:  c.Role,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.UserHandler.InviteToOrganization(c.Context, *state.Organization, flags)
	})
}

type ChangeRoleInOrganizationCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Email        string                `         flag:"" help:"The email of the user to change the role of"`
	Role         string                `         flag:"" help:"The new role of the user"`
}

func (c *ChangeRoleInOrganizationCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.Ignore,
	}

	flags := models.ChangeUserRoleInOrganizationFlags{
		Email: c.Email,
		Role:  c.Role,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.UserHandler.ChangeRoleInOrganization(c.Context, *state.Organization, flags)
	})
}

type UpdateCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Name         string                `         flag:"" help:"The new name of the user"`
}

func (c *UpdateCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.Ignore,
	}

	flags := models.UpdateUserFlags{
		Name: c.Name,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(_ models.CommandState) error {
		err := c.Dependencies.UserHandler.Update(c.Context, flags)
		return err
	})
}
