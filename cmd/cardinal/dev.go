package cardinal

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/magefile/mage/sh"
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/tea/style"
)

const (
	CardinalPort = "3333"
	RedisPort    = "6379"
	WebdisPort   = "7379"
)

/////////////////
// Cobra Setup //
/////////////////

// devCmd runs Cardinal in development mode
// Usage: `world cardinal dev`
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "TODO",
	Long:  `TODO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := os.Chdir("cardinal")
		if err != nil {
			return err
		}

		// Run Redis container
		err = sh.Run("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "-e", "LOCAL_REDIS=true", "--name", "cardinal-dev-redis", "redis")
		if err != nil {
			return err
		}

		// Run Webdis container - this provides a REST wrapper around Redis
		err = sh.Run("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", WebdisPort, WebdisPort), "--link", "cardinal-dev-redis:redis", "--name", "cardinal-dev-webdis", "anapsix/webdis")
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

			if cardinalExecCmd.ProcessState == nil && cardinalExecCmd.Process != nil {
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

// runCardinal runs cardinal in dev mode.
// We run cardinal without docker to make it easier to debug and skip the docker image build step
func runCardinal() (*exec.Cmd, error) {
	fmt.Print(style.CLIHeader("Cardinal", "Running Cardinal in dev mode"), "\n")
	fmt.Println(style.BoldText.Render("Press Ctrl+C to stop"))
	fmt.Println()
	fmt.Println(fmt.Sprintf("Redis: localhost:%s", RedisPort))
	fmt.Println(fmt.Sprintf("Webdis: localhost:%s", WebdisPort))
	fmt.Println(fmt.Sprintf("Cardinal: localhost:%s", CardinalPort))
	fmt.Println()

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

	err := cmd.Start()
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

	err = sh.Run("docker", "rm", "-f", "cardinal-dev-webdis")
	if err != nil {
		fmt.Println("Failed to delete Webdis container automatically")
		fmt.Println("Please delete it manually with `docker rm -f cardinal-dev-webdis`")
		return err
	}

	return nil
}
