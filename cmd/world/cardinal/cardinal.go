package cardinal

import (
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/dependency"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"
)

// BaseCmd is the base command for the cardinal subcommand
// Usage: `world cardinal`
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

func init() {
	// Register subcommands - `world cardinal [subcommand]`
	BaseCmd.AddCommand(startCmd, devCmd, restartCmd, purgeCmd, stopCmd, buildCmd)
	registerConfigAndVerboseFlags(startCmd, devCmd, restartCmd, purgeCmd, stopCmd, buildCmd)
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
