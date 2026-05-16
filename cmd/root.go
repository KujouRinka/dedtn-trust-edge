package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(watchCmd)
}

var appDesc string = "none"

var rootCmd = &cobra.Command{
	Use:   "None",
	Short: appDesc,
	Args:  cobra.ExactArgs(0),
	Run:   runMain,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runMain(cmd *cobra.Command, args []string) {

}
