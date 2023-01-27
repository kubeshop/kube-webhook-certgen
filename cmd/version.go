package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/kubeshop/kube-webhook-certgen/core"
)

var version = &cobra.Command{
	Use:   "version",
	Short: "Prints the CLI version information",
	Run:   versionCmdRun,
}

func versionCmdRun(cmd *cobra.Command, args []string) {
	fmt.Printf("%s\n", core.Version)
	fmt.Printf("%s\n", runtime.Version())
}

func init() {
	rootCmd.AddCommand(version)
}
