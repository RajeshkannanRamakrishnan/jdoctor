package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"strings"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Check Maven/Gradle build health",
	Run: func(cmd *cobra.Command, args []string) {
		results := scanner.ScanBuild()
		for _, res := range results {
			if res.Error != nil {
				fmt.Printf("❌ %s: %v\n", res.Name, res.Error)
			} else {
				// Just grab the first line of version output for brevity
				firstLine := strings.Split(res.Version, "\n")[0]
				fmt.Printf("✔ %s: Healthy (%s)\n", res.Name, firstLine)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
