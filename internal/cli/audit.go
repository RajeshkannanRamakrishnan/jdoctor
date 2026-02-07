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

		var results []scanner.ScanResult

		if len(deps) == 0 {
			fmt.Println("⚠ No dependencies found in pom.xml to scan.")
		} else {
			fmt.Printf("🔍 Scanning %d dependencies against OSV database...\n", len(deps))
			results, err = scanner.ScanVulnerabilities(deps)
			if err != nil {
				fmt.Printf("❌ Vulnerability scan failed: %v\n", err)
				os.Exit(1)
			}

			if len(results) == 0 {
				fmt.Println("✔ No known vulnerabilities found in direct dependencies.")
			} else {
				fmt.Printf("\n🚨 Found vulnerabilities in %d packages:\n", len(results))
				for _, res := range results {
					fmt.Printf("\n📦 %s:%s@%s\n", res.Dependency.GroupId, res.Dependency.ArtifactId, res.Dependency.Version)
					for _, v := range res.Vulns {
						fmt.Printf("   ❌ [%s] %s\n", v.ID, v.Summary)
					}
				}
			}
		}

		// --- 2. Source Code SAST Scan ---
		fmt.Println("\n🔍 Scanning source code for security patterns...")
		cwd, _ := os.Getwd()
		sastVulns, err := scanner.ScanSourceCode(cwd)
		if err != nil {
			fmt.Printf("⚠️ Source scan had errors: %v\n", err)
		} else {
			if len(sastVulns) == 0 {
				fmt.Println("✔ No suspicious coding patterns found in .java files.")
			} else {
				fmt.Printf("\n🚨 Found %d potential security issues in source code:\n", len(sastVulns))
				for _, v := range sastVulns {
					fmt.Printf("   ❌ [%s] %s\n", v.Severity, v.ID)
					fmt.Printf("      File: %s:%d\n", v.File, v.Line)
					fmt.Printf("      Code: %s\n", v.Match)
					fmt.Printf("      -> %s\n\n", v.Description)
				}
			}
		}

		if len(results) > 0 || len(sastVulns) > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
