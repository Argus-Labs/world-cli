package cardinal

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/registry"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/style"
)

type BuildCmd struct {
	LogLevel  string `flag:"" help:"Set the log level for Cardinal"`
	Debug     bool   `flag:"" help:"Enable debugging"`
	Telemetry bool   `flag:"" help:"Enable tracing, metrics, and profiling"`
	Push      string `flag:"" help:"Push your cardinal image to a given image repository"`
	Auth      string `flag:"" help:"Auth token for the given image repository"`
	User      string `flag:"" help:"User for the given image repository"`
	Pass      string `flag:"" help:"Password for the given image repository"`
	RegToken  string `flag:"" help:"Registry token for the given image repository"`
}

func (c *BuildCmd) Run() error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	cfg.Timeout = -1

	if c.LogLevel != "" {
		zeroLogLevel, err := zerolog.ParseLevel(c.LogLevel)
		if err != nil {
			return eris.Errorf("invalid value for flag %s: must be one of (%v)", flagLogLevel, validLogLevels())
		}
		cfg.DockerEnv[DockerCardinalEnvLogLevel] = zeroLogLevel.String()
	}

	if val, exists := cfg.DockerEnv[DockerCardinalEnvLogLevel]; !exists || val == "" {
		// Set default log level to 'info' if log level is not set
		cfg.DockerEnv[DockerCardinalEnvLogLevel] = zerolog.InfoLevel.String()
	} else if _, err := zerolog.ParseLevel(cfg.DockerEnv[DockerCardinalEnvLogLevel]); err != nil {
		// make sure the log level is valid when the flag is not set and using env var from config
		// Error when CARDINAL_LOG_LEVEL is not a valid log level
		return eris.Errorf("invalid value for %s env variable in the config file: must be one of (%v)",
			DockerCardinalEnvLogLevel, validLogLevels())
	}

	// Print out header
	printer.Infoln(style.CLIHeader("Cardinal", ""))
	// Print out service addresses
	printServiceAddress("Redis", cfg.DockerEnv["REDIS_ADDRESS"])
	// this can be changed in code by calling WithPort() on world options, but we have no way to detect that
	printServiceAddress("Cardinal", fmt.Sprintf("localhost:%s", CardinalPort))
	printer.NewLine(2)
	printer.Infoln("Building Cardinal game shard image...")
	printer.Infoln("This may take a few minutes.")

	ctx := context.Background()
	group, ctx := errgroup.WithContext(ctx)

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	services := getCardinalServices(cfg)

	if c.Auth != "" { // FIXME: not sure if this is correct: passed in by then overwritten?
		if (c.User != "" && c.Pass != "") || c.RegToken != "" {
			authConfig := registry.AuthConfig{
				Username:      c.User,
				Password:      c.Pass,
				RegistryToken: c.RegToken,
			}
			c.Auth, _ = registry.EncodeAuthConfig(authConfig)
		}
	}

	// Build the World Engine stack
	group.Go(func() error {
		if err := dockerClient.Build(ctx, c.Push, c.Auth, services...); err != nil {
			return eris.Wrap(err, "Encountered an error with Docker")
		}
		return eris.Wrap(ErrGracefulExit, "Stack terminated")
	})

	// If any of the group's goroutines is terminated non-gracefully, we want to treat it as an error.
	if err := group.Wait(); err != nil && !eris.Is(err, ErrGracefulExit) {
		return err
	}

	return nil
}
