package cardinal

import (
	"context"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
)

type RestartCmd struct {
	Detach bool `flag:"" help:"Run in detached mode"`
	Debug  bool `flag:"" help:"Enable debugging"`
}

func (c *RestartCmd) Run() error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	cfg.Build = true
	cfg.Debug = c.Debug
	cfg.Detach = c.Detach

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	ctx := context.Background()
	err = dockerClient.Restart(ctx, getServices(cfg)...)
	if err != nil {
		return err
	}

	return nil
}
