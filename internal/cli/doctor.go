package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"strings"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run a full health scan",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running full system diagnosis...\n")

		// 1. Env Scan
		env := scanner.ScanEnv()
		if env.Error == nil {
			fmt.Printf("✔ Java %s detected (OS: %s, Arch: %s)\n", env.JavaVersion, env.OS, env.Arch)
		} else {
			fmt.Printf("❌ Java check failed: %v\n", env.Error)
		}

		// 2. Build Scan
		builds := scanner.ScanBuild()
		for _, b := range builds {
			if b.Error == nil {
				// simplistic version info
				firstLine := strings.Split(b.Version, "\n")[0]
				// Truncate if too long
				if len(firstLine) > 50 {
					firstLine = firstLine[:47] + "..."
				}
				fmt.Printf("✔ %s health check passed (%s)\n", b.Name, firstLine)
			} else {
				fmt.Printf("❌ %s check failed: %v\n", b.Name, b.Error)
			}
		}

		// 3. Dependency Scan
		conflicts, err := scanner.ScanDeps()
		if err != nil {
			// If pom.xml is missing, it's not necessarily a fatal error for "doctor", just skip
			if strings.Contains(err.Error(), "pom.xml not found") {
				// silent or minimal info
			} else {
				fmt.Printf("⚠ Dependency scan error: %v\n", err)
			}
		} else {
			if len(conflicts) > 0 {
				fmt.Printf("⚠ %d dependency conflicts found\n", len(conflicts))
			} else {
				fmt.Println("✔ No dependency conflicts found (Maven)")
			}
		}

		fmt.Println("\nDiagnosis complete.")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
