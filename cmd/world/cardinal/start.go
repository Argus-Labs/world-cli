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
	"pkg.world.dev/world-cli/common"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/style"
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

type StartCmd struct {
	Build      bool   `flag:"" help:"Rebuild Docker images before starting"`
	Detach     bool   `flag:"" help:"Run in detached mode"`
	LogLevel   string `flag:"" help:"Set the log level for Cardinal"`
	Debug      bool   `flag:"" help:"Enable delve debugging"`
	Telemetry  bool   `flag:"" help:"Enable tracing, metrics, and profiling"`
	Editor     bool   `flag:"" help:"Run Cardinal Editor, useful for prototyping and debugging"`
	EditorPort string `flag:"" help:"Port for Cardinal Editor"                                  default:"auto"`
}

func (c *StartCmd) Run() error { //nolint:gocognit // this is a naturally complex command
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	cfg.Build = c.Build
	cfg.Debug = c.Debug
	cfg.Detach = c.Detach
	cfg.Telemetry = c.Telemetry
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
	var editorPort int
	if c.Editor { //nolint:nestif // this is not overly complex
		if c.EditorPort == "auto" {
			editorPort, err = common.FindUnusedPort(cePortStart, cePortEnd)
			if err != nil {
				return eris.Wrap(err, "Failed to find an unused port for Cardinal Editor")
			}
		} else {
			editorPort, err = strconv.Atoi(c.EditorPort)
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

	group, ctx := errgroup.WithContext(context.Background())

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	services := getServices(cfg)

	// Start the World Engine stack
	group.Go(func() error {
		if err := dockerClient.Start(ctx, services...); err != nil {
			return eris.Wrap(err, "Encountered an error with Docker")
		}
		return eris.Wrap(ErrGracefulExit, "Stack terminated")
	})

	// Start Cardinal Editor is flag is set to true
	if c.Editor {
		group.Go(func() error {
			if err := startCardinalEditor(ctx, cfg.RootDir, cfg.GameDir, editorPort); err != nil {
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
