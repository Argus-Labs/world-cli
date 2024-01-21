package cardinal

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"pkg.world.dev/world-cli/pkg/logger"
	"pkg.world.dev/world-cli/utils/tea/style"
	"syscall"
	"time"
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
func (c *cardinal) DevCmd() *cobra.Command {
	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Run Cardinal in development mode",
		Long:  `Run Cardinal in development mode`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.SetDebugMode(cmd)

			fmt.Print(style.CLIHeader("Cardinal", "Running Cardinal in dev mode"), "\n")
			fmt.Println(style.BoldText.Render("Press Ctrl+C to stop"))
			fmt.Println()
			fmt.Println(fmt.Sprintf("Redis: localhost:%s", RedisPort))
			fmt.Println(fmt.Sprintf("Cardinal: localhost:%s", CardinalPort))
			fmt.Println()

			// Run Redis container
			err := c.runRedis()
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

			// Run Cardinal
			cardinalExecCmd, err := c.runCardinal()
			if err != nil {
				return err
			}

			// Create a channel to receive termination signals
			signalCh := make(chan os.Signal, 1)
			signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

			// Create a channel to receive errors from the command
			cmdErr := make(chan error, 1)

			go func() {
				err := c.terminal.Wait(cardinalExecCmd)
				cmdErr <- err
			}()

			select {
			case <-signalCh:
				// Shutdown signal received, attempt to gracefully stop the command
				errCleanup := c.cleanup()
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
				errCleanup := c.cleanup()
				if errCleanup != nil {
					return errCleanup
				}
				return nil
			}
		},
	}

	return devCmd
}

// runRedis runs Redis in a Docker container
func (c *cardinal) runRedis() error {
	logger.Println("Starting Redis container...")
	cmd := exec.Command("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name", "cardinal-dev-redis", "redis")
	cmd.Stdout = os.Stdout

	// hide stderr if not in debug mode
	if logger.DebugMode {
		cmd.Stderr = os.Stderr
	}

	_, err := c.terminal.ExecCmd(cmd)
	if err != nil {
		logger.Println("Failed to start Redis container. Retrying after cleanup...")
		cleanupErr := c.cleanup()
		if cleanupErr != nil {
			return err
		}

		_, err := c.terminal.Exec("docker", "run", "-d", "-p", fmt.Sprintf("%s:%s", RedisPort, RedisPort), "--name", "cardinal-dev-redis", "redis")
		if err != nil {
			return err
		}
	}

	return nil
}

// runCardinal runs cardinal in dev mode.
// We run cardinal without docker to make it easier to debug and skip the docker image build step
func (c *cardinal) runCardinal() (*exec.Cmd, error) {
	err := c.terminal.Chdir("cardinal")
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

	// hide stderr if not in debug mode
	if logger.DebugMode {
		cmd.Stderr = os.Stderr
	}

	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	err = cmd.Start()
	if err != nil {
		return cmd, err
	}
	err = cmd.Wait()
	if err != nil {
		return cmd, err
	}

	return cmd, nil
}

// cleanup stops and removes the Redis and Webdis containers
func (c *cardinal) cleanup() error {
	_, err := c.terminal.Exec("docker", "rm", "-f", "cardinal-dev-redis")
	if err != nil {
		logger.Println("Failed to delete Redis container automatically")
		logger.Println("Please delete it manually with `docker rm -f cardinal-dev-redis`")
		return err
	}

	return nil
}
