package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running Java processes with enhanced details",
	Run: func(cmd *cobra.Command, args []string) {
		processes, err := scanner.ScanJavaProcesses()
		if err != nil {
			fmt.Printf("❌ Failed to scan processes: %v\n", err)
			return
		}

		if len(processes) == 0 {
			fmt.Println("No Java processes found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PID\tUPTIME\tNAME\tARGS (Summarized)")
		fmt.Fprintln(w, "---\t------\t----\t-----------------")

		for _, p := range processes {
			// Summarize args to avoid spamming the terminal
			argsSummary := strings.Join(p.Args, " ")
			if len(argsSummary) > 60 {
				argsSummary = argsSummary[:57] + "..."
			}
			
			// Highlight suspicious args if any (simple example)
			if strings.Contains(argsSummary, "-Xdebug") || strings.Contains(argsSummary, "jdwp") {
				argsSummary += " 🐞 (DEBUG)"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.PID, p.Uptime, p.Name, argsSummary)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(psCmd)
}
