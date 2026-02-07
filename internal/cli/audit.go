package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"os"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan project dependencies for security vulnerabilities (CVEs)",
	Long:  `Scans the project's direct dependencies (from pom.xml) against the OSV (Open Source Vulnerabilities) database to identify known security issues.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Parsing project dependencies...")
		deps, err := scanner.GetProjectDependencies()
		if err != nil {
			fmt.Printf("❌ Failed to parse dependencies: %v\n", err)
			return
		}

		if len(deps) == 0 {
			fmt.Println("⚠ No dependencies found in pom.xml to scan.")
			return
		}

		fmt.Printf("🔍 Scanning %d dependencies against OSV database...\n", len(deps))
		results, err := scanner.ScanVulnerabilities(deps)
		if err != nil {
			fmt.Printf("❌ Vulnerability scan failed: %v\n", err)
			os.Exit(1)
		}

		if len(results) == 0 {
			fmt.Println("✔ No known vulnerabilities found in direct dependencies.")
			return
		}

		fmt.Printf("\n🚨 Found vulnerabilities in %d packages:\n", len(results))
		for _, res := range results {
			fmt.Printf("\n📦 %s:%s@%s\n", res.Dependency.GroupId, res.Dependency.ArtifactId, res.Dependency.Version)
			for _, v := range res.Vulns {
				fmt.Printf("   ❌ [%s] %s\n", v.ID, v.Summary)
				// fmt.Printf("      Details: %s\n", v.Details) // Details can be very long
			}
		}
		
		fmt.Println("\nRun 'jdoctor audit' regularly to stay safe!")
		os.Exit(1) // Exit with error code if vulns found
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
