package cardinal

import (
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/common/docker/service"
)

var CardinalCmdPlugin struct {
	Cardinal *CardinalCmd `cmd:"" group:"Cardinal Commands:" help:"Manage your Cardinal game shard"`
}

//nolint:lll, revive // needed to put all the help text in the same line
type CardinalCmd struct {
	Config string `flag:"" type:"existingfile" help:"A TOML config file"`

	Start   *StartCmd   `cmd:"" group:"Cardinal Commands:" help:"Launch your Cardinal game environment"`
	Stop    *StopCmd    `cmd:"" group:"Cardinal Commands:" help:"Gracefully shut down your Cardinal game environment"`
	Restart *RestartCmd `cmd:"" group:"Cardinal Commands:" help:"Restart your Cardinal game environment"`
	Dev     *DevCmd     `cmd:"" group:"Cardinal Commands:" help:"Run Cardinal in fast development mode with hot reloading"`
	Purge   *PurgeCmd   `cmd:"" group:"Cardinal Commands:" help:"Reset your Cardinal game shard to a clean state by removing all data and containers"`
	Build   *BuildCmd   `cmd:"" group:"Cardinal Commands:" help:"Build and package your Cardinal game into production-ready Docker images"`
}

func (c *CardinalCmd) Run() error {
	return dependency.Check(
		dependency.Go,
		dependency.Git,
		dependency.Docker,
		dependency.DockerDaemon,
	)
}

func getServices(cfg *config.Config) []service.Builder {
	services := []service.Builder{service.NakamaDB, service.Redis, service.Cardinal, service.Nakama}
	if cfg.Telemetry && cfg.DockerEnv["NAKAMA_TRACE_ENABLED"] == "true" {
		services = append(services, service.Jaeger)
	}
	if cfg.Telemetry && cfg.DockerEnv["NAKAMA_METRICS_ENABLED"] == "true" {
		services = append(services, service.Prometheus)
	}
	return services
}

func getCardinalServices(_ *config.Config) []service.Builder {
	services := []service.Builder{service.Cardinal}
	return services
}
