package cardinal

import (
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker/service"
)

var CardinalCmdPlugin struct {
	Cardinal *CardinalCmd `cmd:"" group:"Cardinal Commands:" help:"Manage your Cardinal game shard"`
}

type CardinalCmd struct {
	Config string `flag:"" help:"A TOML config file"`

	Start   *StartCmd   `cmd:"" group:"Management Commands:" help:"Launch your Cardinal game environment"`
	Stop    *StopCmd    `cmd:"" group:"Management Commands:" help:"Gracefully shut down your Cardinal game environment"`
	Restart *RestartCmd `cmd:"" group:"Management Commands:" help:"Restart your Cardinal game environment"`
	Dev     *DevCmd     `cmd:"" group:"Management Commands:" help:"Run Cardinal in fast development mode with hot reloading"`
	Purge   *PurgeCmd   `cmd:"" group:"Management Commands:" help:"Reset your Cardinal game shard to a clean state by removing all data and containers"`
	Build   *BuildCmd   `cmd:"" group:"Management Commands:" help:"Build and package your Cardinal game into production-ready Docker images"`
}

/*
// BaseCmd is the base command for the cardinal subcommand.
// Usage: `world cardinal`.

	var BaseCmd = &cobra.Command{
		Use:     "cardinal",
		Short:   "Powerful tools for managing your Cardinal game shard",
		Long:    style.CLIHeader("World CLI — CARDINAL", "Build, run, and manage your Cardinal game shard with ease"),
		GroupID: "core",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return dependency.Check(
				dependency.Go,
				dependency.Git,
				dependency.Docker,
				dependency.DockerDaemon,
			)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmd.Help(); err != nil {
				logger.Fatalf("Failed to execute cardinal command : %s", err.Error())
				return err
			}
			return nil
		},
	}

	func cardinalInit() {
		// Register subcommands - `world cardinal [subcommand]`
		BaseCmd.AddCommand(startCmd, devCmd, restartCmd, purgeCmd, stopCmd, buildCmd)
		registerConfigAndVerboseFlags(startCmd, devCmd, restartCmd, purgeCmd, stopCmd, buildCmd)
	}
*/
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
