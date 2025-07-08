package organization

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

var CmdPlugin struct {
	Organization *Cmd `cmd:"" aliases:"org"  group:"Organization Commands:" help:"Manage your organizations"`
}

type Cmd struct {
	Create  *CreateCmd      `cmd:"" group:"Organization Commands:" help:"Create a new organization"`
	Switch  *SwitchCmd      `cmd:"" group:"Organization Commands:" help:"Switch to an organization"`
	Members *MembersListCmd `cmd:"" group:"Organization Commands:" help:"List members of an organization"`
}

type CreateCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Name         string                `         flag:"" help:"The name of the organization"`
	Slug         string                `         flag:"" help:"The slug of the organization"`
}

func (c *CreateCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedRepoLookup,
		ProjectRequired:      models.NeedRepoLookup,
	}

	flags := models.CreateOrganizationFlags{
		Name: c.Name,
		Slug: c.Slug,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(_ models.CommandState) error {
		_, err := c.Dependencies.OrganizationHandler.Create(c.Context, flags)
		return err
	})
}

type SwitchCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Slug         string                `         flag:"" help:"The slug of the organization to switch to"`
}

func (c *SwitchCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedRepoLookup,
		ProjectRequired:      models.NeedRepoLookup,
	}

	flags := models.SwitchOrganizationFlags{
		Slug: c.Slug,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(_ models.CommandState) error {
		_, err := c.Dependencies.OrganizationHandler.Switch(c.Context, flags)
		return err
	})
}

type MembersListCmd struct {
	Context        context.Context       `kong:"-"`
	Dependencies   cmdsetup.Dependencies `kong:"-"`
	IncludeRemoved bool                  `         flag:"" help:"List removed members"`
}

func (c *MembersListCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.Ignore,
	}

	flags := models.MembersListFlags{
		IncludeRemoved: c.IncludeRemoved,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.OrganizationHandler.MembersList(c.Context, *state.Organization, flags)
	})
}
