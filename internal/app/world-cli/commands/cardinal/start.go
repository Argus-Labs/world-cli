package cardinal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"pkg.world.dev/world-cli/internal/app/world-cli/common"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/config"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/docker"
	"pkg.world.dev/world-cli/internal/app/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/internal/app/world-cli/models"
	"pkg.world.dev/world-cli/internal/pkg/printer"
	"pkg.world.dev/world-cli/internal/pkg/tea/style"
)

const (
	flagBuild     = "build"
	flagDebug     = "debug"
	flagDetach    = "detach"
	flagLogLevel  = "log-level"
	flagEditor    = "editor"
	flagTelemetry = "telemetry"

	// DockerCardinalEnvLogLevel Environment variable name for Docker.
	DockerCardinalEnvLogLevel = "CARDINAL_LOG_LEVEL"
)

var (
	ErrGracefulExit = eris.New("Process gracefully exited")
)

//nolint:gocognit // this is a naturally complex command
func (h *Handler) Start(ctx context.Context, f models.StartCardinalFlags) error {
	cfg, err := config.GetConfig(&f.Config)
	if err != nil {
		return err
	}
	cfg.Build = true
	cfg.Debug = f.Debug
	cfg.Detach = f.Detach
	cfg.Telemetry = f.Telemetry
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
	var editorPort int
	if f.Editor { //nolint:nestif // this is not overly complex
		if f.EditorPort == "auto" {
			editorPort, err = common.FindUnusedPort(cePortStart, cePortEnd)
			if err != nil {
				return eris.Wrap(err, "Failed to find an unused port for Cardinal Editor")
			}
		} else {
			editorPort, err = strconv.Atoi(f.EditorPort)
			if err != nil {
				return eris.Wrap(err, "Failed to convert EditorPort to int")
			}
		}
		printServiceAddress("Cardinal Editor", fmt.Sprintf("localhost:%d", editorPort))
	} else {
		printServiceAddress("Cardinal Editor", "[disabled]")
	}
	printer.NewLine(1)

	printer.Info("Press <ENTER> to continue...")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')

	printer.NewLine(1)
	printer.Infoln("Starting Cardinal game shard...")
	printer.Infoln("This may take a few minutes to rebuild the Docker images.")
	printer.Infoln("Use `world cardinal dev` to run Cardinal faster/easier in development mode.")

	group, groupCtx := errgroup.WithContext(ctx)

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	services := getServices(cfg)

	// Start the World Engine stack
	group.Go(func() error {
		if err := dockerClient.Start(groupCtx, services...); err != nil {
			return eris.Wrap(err, "Encountered an error with Docker")
		}
		return eris.Wrap(ErrGracefulExit, "Stack terminated")
	})

	// Start Cardinal Editor is flag is set to true
	if f.Editor {
		group.Go(func() error {
			if err := startCardinalEditor(groupCtx, cfg.RootDir, cfg.GameDir, editorPort); err != nil {
				return eris.Wrap(err, "Encountered an error with Cardinal Editor")
			}
			return eris.Wrap(ErrGracefulExit, "Cardinal Editor terminated")
		})
	}

	// If any of the group's goroutines is terminated non-gracefully, we want to treat it as an error.
	if err := group.Wait(); err != nil && !eris.Is(err, ErrGracefulExit) {
		return err
	}

	return nil
}

// validLogLevels returns a string of all Valid log levels for zerolog.
func validLogLevels() string {
	return strings.Join([]string{
		zerolog.TraceLevel.String(),
		zerolog.DebugLevel.String(),
		zerolog.InfoLevel.String(),
		zerolog.WarnLevel.String(),
		zerolog.ErrorLevel.String(),
		zerolog.FatalLevel.String(),
		zerolog.PanicLevel.String(),
		zerolog.Disabled.String(),
	}, ", ")
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
