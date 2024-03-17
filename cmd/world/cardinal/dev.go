package cardinal

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/argus-labs/fresh/runner"
	"github.com/magefile/mage/sh"
	"github.com/spf13/cobra"

	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"
)

const (
	CardinalPort = "4040"
	RedisPort    = "6379"

	// flagWatch : Flag for hot reload support
	flagWatch = "watch"
)

// StopChan is used to signal graceful shutdown
var StopChan = make(chan struct{})

// devCmd runs Cardinal in development mode
// Usage: `world cardinal dev`
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run Cardinal in development mode",
	Long:  `Run Cardinal in development mode`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		watch, _ := cmd.Flags().GetBool(flagWatch)
		logger.SetDebugMode(cmd)

		startingMessage := "Running Cardinal in dev mode"
		if watch {
			startingMessage += " with hot reload support"
		}

		fmt.Print(style.CLIHeader("Cardinal", startingMessage), "\n")
		fmt.Println(style.BoldText.Render("Press Ctrl+C to stop"))
		fmt.Println()
		fmt.Printf("Redis: localhost:%s\n", RedisPort)
		fmt.Printf("Cardinal: localhost:%s\n", CardinalPort)
		fmt.Println()

		// Run Redis container
		err := runRedis()
		if err != nil {
			return err
		}

		isRedisRunning := false
		for !isRedisRunning {
			server := fmt.Sprintf("localhost:%s", RedisPort)
			timeout := 2 * time.Second //nolint:gomnd

			conn, err := net.DialTimeout("tcp", server, timeout)
			if err != nil {
				logger.Printf("Failed to connect to Redis server at %s: %s\n", server, err)
				continue
			}
			err = conn.Close()
			if err != nil {
				continue
			}
			isRedisRunning = true
		}

		// Create a channel to receive termination signals
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

		// Run Cardinal Preparation
		err = runCardinalPrep()
		if err != nil {
			return err
		}

		fmt.Println("Starting Cardinal...")
		execCmd, err := runCardinal(watch)
		if err != nil {
			return err
		}

		// Start a goroutine to listen for signals
		go func() {
			<-signalCh
			close(StopChan)
		}()

		// Wait for stop signal
		<-StopChan
		err = stopCardinal(execCmd, watch)
		if err != nil {
			return err
		}

		// Cleanup redis
		errCleanup := cleanup()
		if errCleanup != nil {
			return errCleanup
		}

		return nil

	},
}

/////////////////
// Cobra Setup //
/////////////////

func init() {
	devCmd.Flags().Bool(flagWatch, false, "Dev mode with hot reload support")
}

// runRedis runs Redis in a Docker container
func runRedis() error {
	logger.Println("Starting Redis container...")
	//nolint:gosec // not applicable
	cmd := exec.Command("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name",
		"cardinal-dev-redis", "redis")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		logger.Println("Failed to start Redis container. Retrying after cleanup...")
		cleanupErr := cleanup()
		if cleanupErr != nil {
			return err
		}

		err := sh.Run("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name",
			"cardinal-dev-redis", "redis")
		if err != nil {
			if sh.ExitStatus(err) == 125 { //nolint:gomnd
				fmt.Println("Maybe redis cardinal docker is still up, run 'world cardinal stop' and try again")
				return err
			}
			return err
		}
	}

	return nil
}

// runCardinalPrep preparation for runs cardinal in dev mode.
// We run cardinal without docker to make it easier to debug and skip the docker image build step
func runCardinalPrep() error {
	err := os.Chdir("cardinal")
	if err != nil {
		return errors.New("can't find cardinal directory. Are you in the root of a World Engine project")
	}

	env := map[string]string{
		"REDIS_MODE":     "normal",
		"CARDINAL_PORT":  CardinalPort,
		"REDIS_ADDR":     fmt.Sprintf("localhost:%s", RedisPort),
		"DEPLOY_MODE":    "development",
		"RUNNER_IGNORED": "assets, tmp, vendor",
	}

	for key, value := range env {
		os.Setenv(key, value)
	}
	return nil
}

// runCardinal runs cardinal in dev mode.
// If watch is true, it uses fresh for hot reload support
// Otherwise, it runs cardinal using `go run .`
func runCardinal(watch bool) (*exec.Cmd, error) {
	if watch {
		// using fresh
		go runner.Start()
		return &exec.Cmd{}, nil
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

// stopCardinal stops the cardinal process
// If watch is true, it stops the fresh process
// Otherwise, it stops the cardinal process
func stopCardinal(cmd *exec.Cmd, watch bool) error {
	if watch {
		// using fresh
		runner.Stop()
		return nil
	}

	// stop the cardinal process
	if runtime.GOOS == "windows" {
		err := cmd.Process.Kill()
		if err != nil {
			return err
		}
	} else {
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			return err
		}
	}

	return nil
}

// cleanup stops and removes the Redis and Webdis containers
func cleanup() error {
	err := sh.Run("docker", "rm", "-f", "cardinal-dev-redis")
	if err != nil {
		logger.Println("Failed to delete Redis container automatically")
		logger.Println("Please delete it manually with `docker rm -f cardinal-dev-redis`")
		return err
	}

	return nil
}
