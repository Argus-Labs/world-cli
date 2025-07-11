package main

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/internal/app/world-cli/controllers/cmd_setup"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
)

//nolint:lll // needed to put all the help text in the same line
var CloudCmdPlugin struct {
	Deploy  *DeployCloudCmd  `cmd:"" group:"Cloud Management Commands:" help:"Deploy your World Forge project to a TEST environment in the cloud"`
	Status  *StatusCloudCmd  `cmd:"" group:"Cloud Management Commands:" help:"Check the status of your deployed World Forge project"`
	Promote *PromoteCloudCmd `cmd:"" group:"Cloud Management Commands:" help:"Deploy your game project to a LIVE environment in the cloud"`
	Destroy *DestroyCloudCmd `cmd:"" group:"Cloud Management Commands:" help:"Remove your game project's deployed infrastructure from the cloud"`
	Reset   *ResetCloudCmd   `cmd:"" group:"Cloud Management Commands:" help:"Restart your game project with a clean state"`
	Logs    *LogsCloudCmd    `cmd:"" group:"Cloud Management Commands:" help:"Tail logs for your game project"`
}

type DeployCloudCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Force        bool                  `         flag:"" help:"Force the deployment"`
}

func (c *DeployCloudCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingIDOnly,
		ProjectRequired:      models.NeedExistingData,
	}

	deployType := models.DeploymentTypeDeploy
	if c.Force {
		deployType = models.DeploymentTypeForceDeploy
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.CloudHandler.Deployment(c.Context, state.Organization.ID, *state.Project, deployType)
	})
}

type StatusCloudCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *StatusCloudCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedExistingData,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.CloudHandler.Status(c.Context, *state.Organization, *state.Project)
	})
}

type PromoteCloudCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *PromoteCloudCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingIDOnly,
		ProjectRequired:      models.NeedExistingData,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.CloudHandler.Deployment(
			c.Context,
			state.Organization.ID,
			*state.Project,
			models.DeploymentTypePromote,
		)
	})
}

type DestroyCloudCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *DestroyCloudCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingIDOnly,
		ProjectRequired:      models.NeedExistingData,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.CloudHandler.Deployment(
			c.Context,
			state.Organization.ID,
			*state.Project,
			models.DeploymentTypeDestroy,
		)
	})
}

type ResetCloudCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *ResetCloudCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingIDOnly,
		ProjectRequired:      models.NeedExistingData,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.CloudHandler.Deployment(
			c.Context,
			state.Organization.ID,
			*state.Project,
			models.DeploymentTypeReset,
		)
	})
}

//nolint:lll // needed to put all the help text in the same line
type LogsCloudCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Region       string                `         arg:"" enum:"ap-southeast-1,eu-central-1,us-east-1,us-west-2" default:"us-west-2" optional:"" help:"The region to tail logs for"`
	Env          string                `         arg:"" enum:"test,live"                                       default:"test"      optional:"" help:"The environment to tail logs for"`
}

func (c *LogsCloudCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingIDOnly,
		ProjectRequired:      models.NeedExistingIDOnly,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(_ models.CommandState) error {
		return c.Dependencies.CloudHandler.TailLogs(c.Context, c.Region, c.Env)
	})
}
