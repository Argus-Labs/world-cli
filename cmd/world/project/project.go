package project

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/cmd/world/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
)

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
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Name         string                `         flag:"" help:"The name of the project"`
	Slug         string                `         flag:"" help:"The slug of the project"`
}

func (c *CreateCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedRepoLookup,
	}

	flags := models.CreateProjectFlags{
		Name: c.Name,
		Slug: c.Slug,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		_, err := c.Dependencies.ProjectHandler.Create(c.Context, *state.Organization, flags)
		return err
	})
}

type SwitchCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Slug         string                `         flag:"" help:"The slug of the project to switch to"`
}

func (c *SwitchCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedRepoLookup,
	}

	flags := models.SwitchProjectFlags{
		Slug: c.Slug,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		_, err := c.Dependencies.ProjectHandler.Switch(c.Context, flags, *state.Organization, false)
		return err
	})
}

type UpdateCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Name         string                `         flag:"" help:"The new name of the project"`
	Slug         string                `         flag:"" help:"The new slug of the project"`
	AvatarURL    string                `         flag:"" help:"The new avatar URL of the project" type:"url"`
}

func (c *UpdateCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedExistingData,
	}

	flags := models.UpdateProjectFlags{
		Name: c.Name,
		Slug: c.Slug,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.ProjectHandler.Update(c.Context, *state.Project, *state.Organization, flags)
	})
}

type DeleteCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *DeleteCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedExistingData,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.ProjectHandler.Delete(c.Context, *state.Project)
	})
}
