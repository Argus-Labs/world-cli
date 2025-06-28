package cloud

import (
	"context"

	cmdsetup "pkg.world.dev/world-cli/cmd/internal/controllers/cmd_setup"
	"pkg.world.dev/world-cli/cmd/internal/models"
)

var CmdPlugin struct {
	Cloud *Cmd `cmd:""`
}

//nolint:lll // needed to put all the help text in the same line
type Cmd struct {
	Deploy  *DeployCmd  `cmd:"" group:"Cloud Management Commands:" help:"Deploy your World Forge project to a TEST environment in the cloud"`
	Status  *StatusCmd  `cmd:"" group:"Cloud Management Commands:" help:"Check the status of your deployed World Forge project"`
	Promote *PromoteCmd `cmd:"" group:"Cloud Management Commands:" help:"Deploy your game project to a LIVE environment in the cloud"`
	Destroy *DestroyCmd `cmd:"" group:"Cloud Management Commands:" help:"Remove your game project's deployed infrastructure from the cloud"`
	Reset   *ResetCmd   `cmd:"" group:"Cloud Management Commands:" help:"Restart your game project with a clean state"`
	Logs    *LogsCmd    `cmd:"" group:"Cloud Management Commands:" help:"Tail logs for your game project"`
}

type DeployCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Force        bool                  `         flag:"" help:"Force the deployment"`
}

func (c *DeployCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingIDOnly,
		ProjectRequired:      models.NeedExistingData,
	}

	deployType := DeploymentTypeDeploy
	if c.Force {
		deployType = DeploymentTypeForceDeploy
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.CloudHandler.Deployment(c.Context, state.Organization.ID, *state.Project, deployType)
	})
}

type StatusCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *StatusCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingData,
		ProjectRequired:      models.NeedExistingData,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(state models.CommandState) error {
		return c.Dependencies.CloudHandler.Status(c.Context, *state.Organization, *state.Project)
	})
}

type PromoteCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *PromoteCmd) Run() error {
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
			DeploymentTypePromote,
		)
	})
}

type DestroyCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *DestroyCmd) Run() error {
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
			DeploymentTypeDestroy,
		)
	})
}

type ResetCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
}

func (c *ResetCmd) Run() error {
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
			DeploymentTypeReset,
		)
	})
}

//nolint:lll // needed to put all the help text in the same line
type LogsCmd struct {
	Context      context.Context       `kong:"-"`
	Dependencies cmdsetup.Dependencies `kong:"-"`
	Region       string                `         arg:"" enum:"ap-southeast-1,eu-central-1,us-east-1,us-west-2" default:"us-west-2" optional:"" help:"The region to tail logs for"`
	Env          string                `         arg:"" enum:"test,live"                                       default:"test"      optional:"" help:"The environment to tail logs for"`
}

func (c *LogsCmd) Run() error {
	req := models.SetupRequest{
		LoginRequired:        models.NeedLogin,
		OrganizationRequired: models.NeedExistingIDOnly,
		ProjectRequired:      models.NeedExistingIDOnly,
	}

	return cmdsetup.WithSetup(c.Context, c.Dependencies, req, func(_ models.CommandState) error {
		return c.Dependencies.CloudHandler.TailLogs(c.Context, c.Region, c.Env)
	})
}
