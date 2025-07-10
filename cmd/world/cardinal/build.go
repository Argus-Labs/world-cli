package cardinal

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/registry"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"pkg.world.dev/world-cli/cmd/world/internal/models"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/style"
)

func (h *Handler) Build(ctx context.Context, f models.BuildCardinalFlags) error {
	cfg, err := config.GetConfig(&f.Config)
	if err != nil {
		// No config file found, create a default config
		printer.Infoln("No config file found, creating a default config")

		// Get current working directory for Cardinal v2 projects
		cwd, err := os.Getwd()
		if err != nil {
			return eris.Wrap(err, "Failed to get current working directory")
		}

		// Create a default config
		cfg = &config.Config{
			RootDir:   cwd,
			GameDir:   cwd,
			Detach:    false,
			Build:     true,
			DockerEnv: make(map[string]string),
		}

		cfg.DockerEnv[DockerCardinalEnvLogLevel] = zerolog.DebugLevel.String()
	}
	cfg.Timeout = -1

	if f.LogLevel != "" {
		zeroLogLevel, err := zerolog.ParseLevel(f.LogLevel)
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

	// Set the namespace
	if cfg.DockerEnv["CARDINAL_NAMESPACE"] == "" {
		cfg.DockerEnv["CARDINAL_NAMESPACE"] = "defaultnamespace"
	}
	printer.Infof("Namespace: %s\n", cfg.DockerEnv["CARDINAL_NAMESPACE"])

	group, groupCtx := errgroup.WithContext(ctx)

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	services := getCardinalServices(cfg)

	if f.Auth != "" { // FIXME: not sure if this is correct: passed in by then overwritten?
		if (f.User != "" && f.Pass != "") || f.RegToken != "" {
			authConfig := registry.AuthConfig{
				Username:      f.User,
				Password:      f.Pass,
				RegistryToken: f.RegToken,
			}
			f.Auth, _ = registry.EncodeAuthConfig(authConfig)
		}
	}

	// Build the World Engine stack
	group.Go(func() error {
		if err := dockerClient.Build(groupCtx, f.Push, f.Auth, services...); err != nil {
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
