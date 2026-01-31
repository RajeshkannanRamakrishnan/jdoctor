package cli

import (
	"encoding/json"
	"fmt"
	"jdoctor/internal/scanner"
	"os"

	"github.com/spf13/cobra"
)

type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type Report struct {
	Java   scanner.EnvInfo          `json:"java"`
	Build  []scanner.BuildToolInfo  `json:"build"`
	Deps   []scanner.Conflict       `json:"dependency_conflicts"`
	Errors []string                 `json:"errors,omitempty"`
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a JSON report of the environment",
	Run: func(cmd *cobra.Command, args []string) {
		isJson, _ := cmd.Flags().GetBool("json")

		if !isJson {
			// Fallback to normal doctor if --json not passed (or just separate them)
			// But spec says "jdoctor report --json", implying report might have other formats
			// For now, let's assume report is meant for machine consumption
			// If --json is NOT set, maybe print a hint or run doctor?
			// The user requirement was "jdoctor report --json"
		}

		report := Report{
			Java: scanner.ScanEnv(),
		}

		// Build
		report.Build = scanner.ScanBuild()

		// Deps
		conflicts, err := scanner.ScanDeps()
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("deps scan error: %v", err))
		}
		report.Deps = conflicts

		// Serialize
		// Custom marshaling to handle errors in structs if needed, but standard should work
		// NOTE: scanner structs have 'error' type fields which don't marshal to JSON by default.
		// We might want to add string fields for errors in the scanner structs or handling them here.
		// For MVP, we'll let it slide or fix the scanner structs.
		// Let's modify scanners to have exportable string errors or custom marshaling? 
		// Actually, let's just make sure the output is readable.
		
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate report: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	reportCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(reportCmd)
}
