package cli

import (
	"fmt"
	"jdoctor/internal/scanner"

	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Check for dependency conflicts (Maven only for now)",
	Run: func(cmd *cobra.Command, args []string) {
		conflicts, err := scanner.ScanDeps()
		if err != nil {
			fmt.Printf("⚠️  Error scanning dependencies: %v\n", err)
			return
		}

		if len(conflicts) == 0 {
			fmt.Println("✔ No obvious dependency conflicts found in pom.xml")
		} else {
			fmt.Printf("⚠ Found %d potential conflicts:\n", len(conflicts))
			for _, c := range conflicts {
				fmt.Printf("  - %s: versions %v\n", c.Artifact, c.Versions)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(depsCmd)
}
