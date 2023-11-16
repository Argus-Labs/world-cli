package evm

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"pkg.world.dev/world-cli/common/teacmd"
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
			err = teacmd.DockerStart(true, false, false, 0, teacmd.DockerServiceDA)
			if err != nil {
				fmt.Errorf("error starting %s docker container: %w", teacmd.DockerServiceDA, err)
			}
			return nil
		},
	}
	cmd.Flags().Bool(FlagDoNotSetToken, false, fmt.Sprintf("When set to true, this command will not write to the %s environment variable.", EnvDAAuthToken))
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
