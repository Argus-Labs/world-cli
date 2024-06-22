package cardinal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"pkg.world.dev/world-cli/common"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/teacmd"
	"pkg.world.dev/world-cli/tea/style"
)

const (
	CardinalPort = "4040"
	RedisPort    = "6379"

	// Cardinal Editor Port Range
	cePortStart = 3000
	cePortEnd   = 4000

	// flagPrettyLog Flag that determines whether to run Cardinal with pretty logging (default: true)
	flagPrettyLog = "pretty-log"
)

// devCmd runs Cardinal in development mode
// Usage: `world cardinal dev`
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run Cardinal in development mode",
	Long:  `Run Cardinal in development mode`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		editor, err := cmd.Flags().GetBool(flagEditor)
		if err != nil {
			return err
		}

		prettyLog, err := cmd.Flags().GetBool(flagPrettyLog)
		if err != nil {
			return err
		}

		cfg, err := config.GetConfig(cmd)
		if err != nil {
			return err
		}

		// Print out header
		fmt.Println(style.CLIHeader("Cardinal", ""))

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
		fmt.Println()

		// Start redis, cardinal, and cardinal editor
		// If any of the services terminates, the entire group will be terminated.
		group, ctx := errgroup.WithContext(cmd.Context())
		group.Go(func() error {
			if err := startRedis(ctx); err != nil {
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

func init() {
	registerEditorFlag(devCmd, true)
	devCmd.Flags().Bool(flagPrettyLog, true, "Run Cardinal with pretty logging")
}

//////////////////////
// Cardinal Helpers //
//////////////////////

// startCardinalDevMode runs cardinal in dev mode.
// If watch is true, it uses fresh for hot reload support
// Otherwise, it runs cardinal using `go run .`
func startCardinalDevMode(ctx context.Context, cfg *config.Config, prettyLog bool) error {
	fmt.Println("Starting Cardinal...")
	fmt.Println(style.BoldText.Render("Press Ctrl+C to stop\n"))

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

// startRedis runs Redis in a Docker container
func startRedis(ctx context.Context) error {
	// Create an error group for managing redis lifecycle
	group := new(errgroup.Group)

	// Start Redis container
	group.Go(func() error {
		if err := teacmd.DockerStart(&config.Config{Detach: true, Build: false},
			[]teacmd.DockerService{teacmd.DockerServiceRedis}); err != nil {
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
		if err := teacmd.DockerStop([]teacmd.DockerService{teacmd.DockerServiceRedis}); err != nil {
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
	fmt.Println(serviceStr + arrowStr + addressStr)
}
