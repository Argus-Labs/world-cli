package main

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/internal/app/world-cli/controllers/cmd_setup"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

var ProjectCmdPlugin struct {
	Project *ProjectCmd `cmd:"" aliases:"proj" group:"Project Commands:"      help:"Manage your projects"`
}

type ProjectCmd struct {
	Create *CreateProjectCmd `cmd:"" group:"Project Commands:" help:"Create a new project"`
	Switch *SwitchProjectCmd `cmd:"" group:"Project Commands:" help:"Switch to a different project"`
	Update *UpdateProjectCmd `cmd:"" group:"Project Commands:" help:"Update your project"`
	Delete *DeleteProjectCmd `cmd:"" group:"Project Commands:" help:"Delete your project"`
}

type CreateProjectCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Name         string                `         flag:"" help:"The name of the project"`
	Slug         string                `         flag:"" help:"The slug of the project"`
}

func (c *CreateProjectCmd) Run() error {
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

type SwitchProjectCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Slug         string                `         flag:"" help:"The slug of the project to switch to"`
}

func (c *SwitchProjectCmd) Run() error {
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

type UpdateProjectCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Name         string                `         flag:"" help:"The new name of the project"`
	Slug         string                `         flag:"" help:"The new slug of the project"`
	AvatarURL    string                `         flag:"" help:"The new avatar URL of the project" type:"url"`
}

func (c *UpdateProjectCmd) Run() error {
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

type DeleteProjectCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *DeleteProjectCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedExistingData,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.ProjectHandler.Delete(c.Context, *state.Project)
	})
}
