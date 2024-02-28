package cardinal

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"pkg.world.dev/world-cli/common/logger"
	"pkg.world.dev/world-cli/tea/style"

	"github.com/argus-labs/fresh/runner"
	"github.com/magefile/mage/sh"
	"github.com/spf13/cobra"
)

const (
	CardinalPort = "4040"
	RedisPort    = "6379"

	// flagWatch : Flag for hot reload support
	flagWatch = "watch"
)

// StopChan is used to signal graceful shutdown
var StopChan = make(chan struct{})

/////////////////
// Cobra Setup //
/////////////////

func init() {
	devCmd.Flags().Bool(flagWatch, false, "Dev mode with hot reload support")
}

// devCmd runs Cardinal in development mode
// Usage: `world cardinal dev`
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run Cardinal in development mode",
	Long:  `Run Cardinal in development mode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		watch, _ := cmd.Flags().GetBool(flagWatch)
		logger.SetDebugMode(cmd)

		startingMessage := "Running Cardinal in dev mode"
		if watch {
			startingMessage += " with hot reload support"
		}

		fmt.Print(style.CLIHeader("Cardinal", startingMessage), "\n")
		fmt.Println(style.BoldText.Render("Press Ctrl+C to stop"))
		fmt.Println()
		fmt.Println(fmt.Sprintf("Redis: localhost:%s", RedisPort))
		fmt.Println(fmt.Sprintf("Cardinal: localhost:%s", CardinalPort))
		fmt.Println()

		// Run Redis container
		err := runRedis()
		if err != nil {
			return err
		}

		isRedisRunning := false
		for !isRedisRunning {
			server := fmt.Sprintf("localhost:%s", RedisPort)
			timeout := 2 * time.Second

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
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

		// Run Cardinal Preparation
		err = runCardinalPrep(watch)
		if err != nil {
			return err
		}

		fmt.Println("Starting Cardinal...")
		go runner.Start()

		// Start a goroutine to listen for signals
		go func() {
			<-signalCh
			close(StopChan)
		}()

		// Wait for stop signal
		<-StopChan
		runner.Stop() // Stop Cardinal

		// Cleanup redis
		errCleanup := cleanup()
		if errCleanup != nil {
			return errCleanup
		}

		return nil

	},
}

// runRedis runs Redis in a Docker container
func runRedis() error {
	logger.Println("Starting Redis container...")
	cmd := exec.Command("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name", "cardinal-dev-redis", "redis")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		logger.Println("Failed to start Redis container. Retrying after cleanup...")
		cleanupErr := cleanup()
		if cleanupErr != nil {
			return err
		}

		err := sh.Run("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name", "cardinal-dev-redis", "redis")
		if err != nil {
			if sh.ExitStatus(err) == 125 {
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
func runCardinalPrep(watch bool) error {
	err := os.Chdir("cardinal")
	if err != nil {
		return errors.New("can't find cardinal directory. Are you in the root of a World Engine project")
	}

	runnerIgnored := "."
	if watch {
		runnerIgnored = "assets, tmp, vendor"
	}

	env := map[string]string{
		"REDIS_MODE":     "normal",
		"CARDINAL_PORT":  CardinalPort,
		"REDIS_ADDR":     fmt.Sprintf("localhost:%s", RedisPort),
		"DEPLOY_MODE":    "development",
		"RUNNER_IGNORED": runnerIgnored,
	}

	for key, value := range env {
		os.Setenv(key, value)
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
