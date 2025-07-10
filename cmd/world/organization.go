package main

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/internal/app/world-cli/controllers/cmd_setup"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

var OrganizationCmdPlugin struct {
	Organization *OrganizationCmd `cmd:"" aliases:"org"  group:"Organization Commands:" help:"Manage your organizations"`
}

type OrganizationCmd struct {
	Create  *CreateOrganizationCmd `cmd:"" group:"Organization Commands:" help:"Create a new organization"`
	Switch  *SwitchOrganizationCmd `cmd:"" group:"Organization Commands:" help:"Switch to an organization"`
	Members *MembersListCmd        `cmd:"" group:"Organization Commands:" help:"List members of an organization"`
}

type CreateOrganizationCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Name         string                `         flag:"" help:"The name of the organization"`
	Slug         string                `         flag:"" help:"The slug of the organization"`
}

func (c *CreateOrganizationCmd) Run() error {
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

type SwitchOrganizationCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Slug         string                `         flag:"" help:"The slug of the organization to switch to"`
}

func (c *SwitchOrganizationCmd) Run() error {
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
