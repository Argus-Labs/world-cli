package cardinal

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"pkg.world.dev/world-cli/common"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/docker"
	"pkg.world.dev/world-cli/common/docker/service"
	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/common/printer"
	"pkg.world.dev/world-cli/tea/style"
)

const (
	CardinalPort = "4040"
	RedisPort    = "6379"

	// Cardinal Editor Port Range.
	cePortStart = 3000
	cePortEnd   = 4000

	// flagPrettyLog Flag that determines whether to run Cardinal with pretty logging (default: true).
	flagPrettyLog = "pretty-log"
)

type DevCmd struct {
	Editor bool `flag:"" help:"Enable Cardinal Editor"`
}

func (c *DevCmd) Run() error {
	_, err := config.GetConfig()
	if err != nil {
		return err
	}
	return nil
}

// Usage: `world cardinal dev`.
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run Cardinal in fast development mode with hot reloading",
	Long: `Launch Cardinal in development mode for rapid iteration and testing.
	
This mode runs Cardinal directly from your source code, providing:
- Faster startup times than the standard Docker deployment
- Immediate feedback on code changes
- Optional Cardinal Editor integration for visual development`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		editor, err := cmd.Flags().GetBool(flagEditor)
		if err != nil {
			return err
		}

		prettyLog, err := cmd.Flags().GetBool(flagPrettyLog)
		if err != nil {
			return err
		}

		cfg, err := config.GetConfig()
		if err != nil {
			return err
		}

		// Print out header
		printer.Infoln(style.CLIHeader("Cardinal", ""))

		// Print out service addresses
		printServiceAddress("Redis", fmt.Sprintf("localhost:%s", RedisPort))
		printServiceAddress("Cardinal", fmt.Sprintf("localhost:%s", CardinalPort))
		var port int
		if editor {
			port, err = common.FindUnusedPort(cePortStart, cePortEnd)
			if err != nil {
				return eris.Wrap(err, "Failed to find an unused port for Cardinal Editor")
			}
			printServiceAddress("Cardinal Editor", fmt.Sprintf("localhost:%d", port))
		} else {
			printServiceAddress("Cardinal Editor", "[disabled]")
		}
		printer.NewLine(1)

		// Start redis, cardinal, and cardinal editor
		// If any of the services terminates, the entire group will be terminated.
		group, ctx := errgroup.WithContext(cmd.Context())
		group.Go(func() error {
			if err := startRedis(ctx, cfg); err != nil {
				return eris.Wrap(err, "Encountered an error with Redis")
			}
			return eris.Wrap(ErrGracefulExit, "Redis terminated")
		})
		group.Go(func() error {
			if err := startCardinalDevMode(ctx, cfg, prettyLog); err != nil {
				return eris.Wrap(err, "Encountered an error with Cardinal")
			}
			return eris.Wrap(ErrGracefulExit, "Cardinal terminated")
		})
		if editor {
			group.Go(func() error {
				if err := startCardinalEditor(ctx, cfg.RootDir, cfg.GameDir, port); err != nil {
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
	},
}

/////////////////
// Cobra Setup //
/////////////////

func devCmdInit() {
	//registerEditorFlag(devCmd, true)
	devCmd.Flags().Bool(flagPrettyLog, true, "Run Cardinal with pretty logging")
}

//////////////////////
// Cardinal Helpers //
//////////////////////

// Otherwise, it runs cardinal using `go run .`.
func startCardinalDevMode(ctx context.Context, cfg *config.Config, prettyLog bool) error { //nolint:gocognit
	printer.Infoln("Starting Cardinal...")
	printer.Infoln(style.BoldText.Render("Press Ctrl+C to stop"))
	printer.NewLine(1)

	// Check and wait until Redis is running and is available in the expected port
	isRedisHealthy := false
	for !isRedisHealthy {
		// using select to allow for context cancellation
		select {
		case <-ctx.Done():
			return eris.Wrap(ctx.Err(), "Context canceled")
		default:
			redisAddress := fmt.Sprintf("localhost:%s", RedisPort)
			conn, err := net.DialTimeout("tcp", redisAddress, time.Second)
			if err != nil {
				logger.Printf("Failed to connect to Redis at %s: %s\n", redisAddress, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// Cleanup connection
			if err := conn.Close(); err != nil {
				continue
			}

			isRedisHealthy = true
		}
	}

	// Move into the cardinal directory
	if err := os.Chdir(filepath.Join(cfg.RootDir, cfg.GameDir)); err != nil {
		return eris.New("Unable to find cardinal directory. Are you in the project root?")
	}

	// Set world.toml environment variables
	if err := common.WithEnv(cfg.DockerEnv); err != nil {
		return eris.Wrap(err, "Failed to set world.toml environment variables")
	}

	// Set dev mode environment variables
	if err := common.WithEnv(
		map[string]string{
			"RUNNER_IGNORED":      "assets, tmp, vendor",
			"CARDINAL_PRETTY_LOG": strconv.FormatBool(prettyLog),
		},
	); err != nil {
		return eris.Wrap(err, "Failed to set dev mode environment variables")
	}

	// Create an error group for managing cardinal lifecycle
	group, ctx := errgroup.WithContext(ctx)

	// Run cardinal
	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	group.Go(func() error {
		if err := cmd.Start(); err != nil {
			return eris.Wrap(err, "Failed to start Cardinal")
		}

		if err := cmd.Wait(); err != nil {
			return err
		}
		return nil
	})

	// Goroutine to handle termination
	// There are two ways that a termination sequence can be triggered:
	// 1) The cardinal goroutine returns a non-nil error
	// 2) The parent context is canceled for whatever reason.
	group.Go(func() error {
		<-ctx.Done()

		// No need to do anything if cardinal already exited or is not running
		if cmd.ProcessState == nil || cmd.ProcessState.Exited() {
			return nil
		}

		if runtime.GOOS == "windows" {
			// Sending interrupt signal is not supported in Windows
			if err := cmd.Process.Kill(); err != nil {
				return err
			}
		} else {
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				return err
			}
		}

		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}

///////////////////
// Redis Helpers //
///////////////////

// startRedis runs Redis in a Docker container.
func startRedis(ctx context.Context, cfg *config.Config) error {
	// Create an error group for managing redis lifecycle
	group := new(errgroup.Group)

	// Create docker client
	dockerClient, err := docker.NewClient(cfg)
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	// Create context with cancel
	ctx, cancel := context.WithCancel(ctx)

	// Start Redis container
	group.Go(func() error {
		cfg.Detach = true
		if err := dockerClient.Start(ctx, service.Redis); err != nil {
			cancel()
			return eris.Wrap(err, "Encountered an error with Redis")
		}
		return nil
	})

	// Goroutine to handle termination
	// There are two ways that a termination sequence can be triggered:
	// 1) The redis start goroutine returns a non-nil error
	// 2) The parent context is canceled for whatever reason.
	group.Go(func() error {
		<-ctx.Done()
		// Using context background because cmd context is already done
		if err := dockerClient.Stop(context.Background(), service.Redis); err != nil {
			return err
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}

///////////////////
// Utils Helpers //
///////////////////

func printServiceAddress(service string, address string) {
	serviceStr := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(service)
	arrowStr := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render(" → ")
	addressStr := lipgloss.NewStyle().Render(address)
	printer.Infoln(serviceStr + arrowStr + addressStr)
}
