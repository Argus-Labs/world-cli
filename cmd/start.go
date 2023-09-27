/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"pkg.world.dev/world-cli/utils"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts everything",
	Long:  `Starts Nakama, Cardinal, Redis and Postgresql`,
	Run: func(cmd *cobra.Command, args []string) {
		utils.RunShellCmd("cd cardinal && go get && go mod vendor && cd ..", true, false)
		utils.RunShellCmd("cd cardinal && go get && go mod vendor && cd ..", true, false)
		utils.RunShellCmd("docker compose up", true, true)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
