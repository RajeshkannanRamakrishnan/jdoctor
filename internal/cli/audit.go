package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"os"
	"strings"

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
						if severity := formatSeverity(v); severity != "" {
							fmt.Printf("      Severity: %s\n", severity)
						}
						if fixed := extractFixedVersions(v); len(fixed) > 0 {
							fmt.Printf("      Fixed Versions: %s\n", strings.Join(fixed, ", "))
						}
						if refs := extractReferenceURLs(v); len(refs) > 0 {
							fmt.Println("      References:")
							for _, r := range refs {
								fmt.Printf("        - %s\n", r)
							}
						}
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

func formatSeverity(v scanner.Vulnerability) string {
	seen := make(map[string]bool)
	var items []string

	addSeverity := func(s scanner.Severity) {
		if s.Type == "" && s.Score == "" {
			return
		}
		label := s.Type
		if label == "" {
			label = "Severity"
		}
		value := s.Score
		if value == "" {
			return
		}
		entry := fmt.Sprintf("%s: %s", label, value)
		if !seen[entry] {
			seen[entry] = true
			items = append(items, entry)
		}
	}

	for _, s := range v.Severity {
		addSeverity(s)
	}
	if len(items) == 0 {
		for _, a := range v.Affected {
			for _, s := range a.Severity {
				addSeverity(s)
			}
		}
	}

	return strings.Join(items, ", ")
}

func extractFixedVersions(v scanner.Vulnerability) []string {
	seen := make(map[string]bool)
	var fixed []string
	for _, a := range v.Affected {
		for _, r := range a.Ranges {
			for _, e := range r.Events {
				if e.Fixed == "" {
					continue
				}
				if !seen[e.Fixed] {
					seen[e.Fixed] = true
					fixed = append(fixed, e.Fixed)
				}
			}
		}
	}
	return fixed
}

func extractReferenceURLs(v scanner.Vulnerability) []string {
	seen := make(map[string]bool)
	var refs []string
	for _, r := range v.References {
		if r.URL == "" {
			continue
		}
		if !seen[r.URL] {
			seen[r.URL] = true
			refs = append(refs, r.URL)
		}
	}
	return refs
}
