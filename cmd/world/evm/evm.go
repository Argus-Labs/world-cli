package evm

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"pkg.world.dev/world-cli/common/teacmd"
	"time"
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
	)
	return evmRootCmd
}

const (
	FlagDoNotSetToken = "dont-set-token"
	FlagDAAuthToken   = "da-auth-token"
	EnvDAAuthToken    = "DA_AUTH_TOKEN"
)

func StartDA() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start-da",
		Short: "Start the data availability layer client.",
		Long:  fmt.Sprintf("Start the data availability layer client. This command will automatically set the %s environment variable unless --%s is used.", EnvDAAuthToken, FlagDoNotSetToken),
		RunE: func(cmd *cobra.Command, args []string) error {
			dontSetToken, err := cmd.Flags().GetBool(FlagDoNotSetToken)
			if err != nil {
				return err
			}
			daService := teacmd.DockerServiceDA
			fmt.Println("starting DA docker service...")
			err = teacmd.DockerStart(true, false, true, -1, daService)
			if err != nil {
				fmt.Errorf("error starting %s docker container: %w", daService, err)
			}
			time.Sleep(3 * time.Second)
			fmt.Println("started DA service...")
			if !dontSetToken { // yes, we want to set it!
				fmt.Println("attempting to set DA token...")
				daToken, err := getDAToken()
				if err != nil {
					return err
				}
				err = os.Setenv(EnvDAAuthToken, daToken)
				if err != nil {
					return fmt.Errorf("error setting environment variable %q to %q: %w", EnvDAAuthToken, daToken, err)
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool(FlagDoNotSetToken, false, fmt.Sprintf("When set to true, this command will not write to the %s environment variable.", EnvDAAuthToken))
	return cmd
}

func StartEVM() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the EVM base shard. Use --da-auth-token to pass in an auth token directly.",
		Long:  "Start the EVM base shard. Requires connection to celestia DA.",
		RunE: func(cmd *cobra.Command, args []string) error {
			daToken, err := cmd.Flags().GetString(FlagDAAuthToken)
			if err != nil {
				return err
			}
			if daToken != "" {
				os.Setenv(EnvDAAuthToken, daToken)
			}
			if os.Getenv(EnvDAAuthToken) == "" {
				return fmt.Errorf("the environment variable %q was not set and the --%s flag was not used. "+
					"please supply the token by either the environment variable, or the flag",
					EnvDAAuthToken, FlagDAAuthToken)
			}
			err = teacmd.DockerStart(true, false, false, 0, teacmd.DockerServiceEVM)
			if err != nil {
				fmt.Errorf("error starting %s docker container: %w", teacmd.DockerServiceEVM, err)
			}
			return nil
		},
	}
	cmd.Flags().String(FlagDAAuthToken, "", "DA Auth Token that allows the rollup to communicate with the Celestia client.")
	return cmd
}

func getDAToken() (string, error) {
	// Create a new command
	cmd := exec.Command("docker", "logs", "celestia_devnet")
	retry := 0
	for ; retry < 10; func() {
		time.Sleep(2 * time.Second)
		retry++
	}() {
		fmt.Println("attempting to get DA token...")
		// Create a buffer to store the command output
		var outBuffer bytes.Buffer
		cmd.Stdout = &outBuffer

		// Run the command
		err := cmd.Run()
		if err != nil {
			fmt.Println("error running command docker logs: ", err)
			continue
		}

		// Convert the buffer to string
		output := outBuffer.String()

		// Find the line containing CELESTIA_NODE_AUTH_TOKEN
		var authTokenLine string
		lines := bytes.Split([]byte(output), []byte("\n"))
		for i, line := range lines {
			if bytes.Contains(line, []byte("CELESTIA_NODE_AUTH_TOKEN")) {
				// Get the next 5 lines after the match
				endIndex := i + 6
				if endIndex > len(lines) {
					endIndex = len(lines)
				}
				authTokenLines := lines[i:endIndex]

				// Concatenate the lines to get the final output
				authTokenLine = string(bytes.Join(authTokenLines, []byte("\n")))
				break
			}
		}

		// Print the final output
		if authTokenLine != "" {
			fmt.Println("token found")
			return authTokenLine, nil
		}
		fmt.Println("failed... trying again")
	}
	return "", fmt.Errorf("timed out while getting DA token")
}
