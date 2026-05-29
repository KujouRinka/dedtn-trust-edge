package cmd

import (
	"github.com/spf13/cobra"
)

var draftCmd = &cobra.Command{
	Use: "draft",
	Run: draftMain,
}

func draftMain(cmd *cobra.Command, args []string) {
}
