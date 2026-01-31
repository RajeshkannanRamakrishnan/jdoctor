package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jdoctor",
	Short: "jdoctor - A CLI tool to diagnose Java developer environments",
	Long:  `jdoctor helps developers quickly identify configuration problems, version mismatches, and build health issues in their Java environment.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
