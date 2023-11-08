package cardinal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"pkg.world.dev/world-cli/tea/style"
	"syscall"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/spf13/cobra"
)

const (
	CardinalPort = "3333"
	RedisPort    = "6379"
)

/////////////////
// Cobra Setup //
/////////////////

// devCmd runs Cardinal in development mode
// Usage: `world cardinal dev`
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run Cardinal in development mode",
	Long:  `Run Cardinal in development mode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(style.CLIHeader("Cardinal", "Running Cardinal in dev mode"), "\n")
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

		// Run Cardinal
		cardinalExecCmd, err := runCardinal()
		if err != nil {
			return err
		}

		// Create a channel to receive termination signals
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

		// Create a channel to receive errors from the command
		cmdErr := make(chan error, 1)

		go func() {
			err := cardinalExecCmd.Wait()
			cmdErr <- err
		}()

		select {
		case <-signalCh:
			// Shutdown signal received, attempt to gracefully stop the command
			errCleanup := cleanup()
			if errCleanup != nil {
				return errCleanup
			}

			isProcessRunning := func(cmd *exec.Cmd) bool {
				return cardinalExecCmd.ProcessState == nil && cardinalExecCmd.Process != nil
			}

			//wait up to 10 seconds for it to quit.
			for i := 0; i < 10; i++ {
				if isProcessRunning(cardinalExecCmd) {
					time.Sleep(1 * time.Second)
				} else {
					break
				}
			}
			if isProcessRunning(cardinalExecCmd) {
				err = cardinalExecCmd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					return err
				}
			}

			return nil

		case err := <-cmdErr:
			fmt.Println(err)
			errCleanup := cleanup()
			if errCleanup != nil {
				return errCleanup
			}
			return nil
		}
	},
}

// runRedis runs Redis in a Docker container
func runRedis() error {
	fmt.Println("Starting Redis container...")
	cmd := exec.Command("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name", "cardinal-dev-redis", "redis")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Println("Failed to start Redis container. Retrying after cleanup...")
		cleanupErr := cleanup()
		if cleanupErr != nil {
			return err
		}

		err := sh.Run("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name", "cardinal-dev-redis", "redis")
		if err != nil {
			return err
		}
	}

	return nil
}

// runCardinal runs cardinal in dev mode.
// We run cardinal without docker to make it easier to debug and skip the docker image build step
func runCardinal() (*exec.Cmd, error) {
	err := os.Chdir("cardinal")
	if err != nil {
		return nil, errors.New("can't find cardinal directory. Are you in the root of a World Engine project")
	}

	env := map[string]string{
		"REDIS_MODE":    "normal",
		"CARDINAL_PORT": CardinalPort,
		"REDIS_ADDR":    fmt.Sprintf("localhost:%s", RedisPort),
		"DEPLOY_MODE":   "development",
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	err = cmd.Start()
	if err != nil {
		return cmd, err
	}

	return cmd, nil
}

// cleanup stops and removes the Redis and Webdis containers
func cleanup() error {
	err := sh.Run("docker", "rm", "-f", "cardinal-dev-redis")
	if err != nil {
		fmt.Println("Failed to delete Redis container automatically")
		fmt.Println("Please delete it manually with `docker rm -f cardinal-dev-redis`")
		return err
	}

	return nil
}
