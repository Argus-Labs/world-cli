package evm

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/common/config"
	"pkg.world.dev/world-cli/common/tea_cmd"
)

func EVMCmds() *cobra.Command {
	evmRootCmd := &cobra.Command{
		Use:   "evm",
		Short: "EVM base shard commands.",
		Long:  "Commands for provisioning the EVM Base Shard.",
	}
	evmRootCmd.AddGroup(&cobra.Group{
		ID:    "EVM",
		Title: "EVM Base Shard Commands",
	})
	evmRootCmd.AddCommand(
		StartEVM(),
		StartDA(),
		StopAll(),
	)
	return evmRootCmd
}

const (
	FlagDAAuthToken = "da-auth-token"
	EnvDAAuthToken  = "DA_AUTH_TOKEN"
)

func services(s ...tea_cmd.DockerService) []tea_cmd.DockerService {
	return s
}

func StartDA() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start-da",
		Short: "Start the data availability layer client.",
		Long:  fmt.Sprintf("Start the data availability layer client."),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig(cmd)
			if err != nil {
				return err
			}

			// Has the DA_AUTH_TOKEN parameter been set in the config?
			isDAAuthTokenSet := len(cfg.DockerEnv[EnvDAAuthToken]) > 0

			cfg.Build = true
			cfg.Debug = false
			cfg.Detach = true
			cfg.Timeout = -1
			daService := tea_cmd.DockerServiceDA
			fmt.Println("starting DA docker service...")
			err = tea_cmd.DockerStart(cfg, services(daService))
			if err != nil {
				fmt.Errorf("error starting %s docker container: %w", daService, err)
			}
			// TODO: Can this be replaced with a health check in the docker-compose file?
			time.Sleep(3 * time.Second)
			fmt.Println("started DA service...")

			if !isDAAuthTokenSet {
				fmt.Println("DA token has not been set in the config file.")
				fmt.Println("attempting to get the DA token...")
				authTokenLog, daToken, err := getDAToken()
				if err != nil {
					return err
				}
				fmt.Println(authTokenLog)
				fmt.Println("To skip this check in the future, add the following line to the [evm] section of your config file:")
				fmt.Printf("%s=%q\n", EnvDAAuthToken, daToken)
			}
			return nil
		},
	}
	return cmd
}

func StartEVM() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the EVM base shard. Use --da-auth-token to pass in an auth token directly.",
		Long:  "Start the EVM base shard. Requires connection to celestia DA.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.GetConfig(cmd)
			if err != nil {
				return err
			}

			daToken, err := cmd.Flags().GetString(FlagDAAuthToken)
			if err != nil {
				return err
			}
			fmt.Println("start", cfg.DockerEnv[EnvDAAuthToken])
			if daToken != "" {
				cfg.DockerEnv[EnvDAAuthToken] = daToken
			}
			fmt.Println("after", cfg.DockerEnv[EnvDAAuthToken])

			if cfg.DockerEnv[EnvDAAuthToken] == "" {
				return fmt.Errorf("the DA auth token was not found in the config at a[evm].%s, nor set via the"+
					"--%s flag. please add the token to the config file or use the flag", EnvDAAuthToken, FlagDAAuthToken)
			}

			cfg.Build = true
			cfg.Debug = false
			cfg.Detach = false
			cfg.Timeout = 0

			err = tea_cmd.DockerStart(cfg, services(tea_cmd.DockerServiceEVM))
			if err != nil {
				fmt.Errorf("error starting %s docker container: %w", tea_cmd.DockerServiceEVM, err)
			}
			return nil
		},
	}
	cmd.Flags().String(FlagDAAuthToken, "", "DA Auth Token that allows the rollup to communicate with the Celestia client.")
	return cmd
}

func StopAll() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the EVM base shard and DA layer client.",
		Long:  "Stop the EVM base shard and data availability layer client if they are running.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tea_cmd.DockerStop(services(tea_cmd.DockerServiceEVM, tea_cmd.DockerServiceDA))
		},
	}
	return cmd
}

func getDAToken() (authTokenLog, token string, err error) {
	// Create a new command
	cmd := exec.Command("docker", "logs", "celestia_devnet")
	retry := 0
	for ; retry < 10; func() {
		time.Sleep(2 * time.Second)
		retry++
	}() {
		fmt.Println("attempting to get DA token...")

		output, err := cmd.Output()
		if err != nil {
			fmt.Println("error running command docker logs: ", err)
			continue
		}

		// Find the line containing CELESTIA_NODE_AUTH_TOKEN
		lines := bytes.Split(output, []byte("\n"))
		for i, line := range lines {
			if bytes.Contains(line, []byte("CELESTIA_NODE_AUTH_TOKEN")) {
				// Get the next 5 lines after the match
				endIndex := i + 6
				if endIndex > len(lines) {
					endIndex = len(lines)
				}
				authTokenLines := lines[i:endIndex]
				fmt.Println("I have some lines", len(authTokenLines))
				for _, X := range authTokenLines {
					fmt.Println("->", string(X))
				}

				// Concatenate the lines to get the final output
				authTokenLog = string(bytes.Join(authTokenLines, []byte("\n")))
				// The last line is the actual token
				token = string(authTokenLines[len(authTokenLines)-1])
				break
			}
		}

		// Print the final output
		if authTokenLog != "" {
			fmt.Println("token found")
			return authTokenLog, token, nil
		}
		fmt.Println("failed... trying again")
	}
	return "", "", fmt.Errorf("timed out while getting DA token")
}
