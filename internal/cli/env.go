package cli

import (
	"fmt"
	"jdoctor/internal/scanner"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Detects Java version, OS, and Architecture",
	Run: func(cmd *cobra.Command, args []string) {
		info := scanner.ScanEnv()
		fmt.Printf("OS:   %s\n", info.OS)
		fmt.Printf("Arch: %s\n", info.Arch)
		if info.Error != nil {
			fmt.Printf("Java: %v\n", info.Error)
		} else {
			fmt.Printf("Java: %s\n", info.JavaVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
