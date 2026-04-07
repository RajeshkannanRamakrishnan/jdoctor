package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"strings"

	"github.com/spf13/cobra"
)

var jdkCmd = &cobra.Command{
	Use:   "jdk",
	Short: "Inspect installed JDKs and active Java configuration",
	Long:  `Detects installed JDKs, the active JAVA_HOME, the java resolved from PATH, and common version or path mismatches.`,
	Run: func(cmd *cobra.Command, args []string) {
		result := scanner.ScanJDKs()

		fmt.Println("JDK Configuration")
		fmt.Println("")

		if result.JavaHome != "" {
			fmt.Printf("JAVA_HOME:      %s\n", result.JavaHome)
			if result.JavaHomeResolved != "" && result.JavaHomeResolved != result.JavaHome {
				fmt.Printf("JAVA_HOME real: %s\n", result.JavaHomeResolved)
			}
		} else {
			fmt.Println("JAVA_HOME:      (not set)")
		}

		if result.PathJava != "" {
			fmt.Printf("PATH java:      %s\n", result.PathJava)
			if result.PathJavaResolved != "" && result.PathJavaResolved != result.PathJava {
				fmt.Printf("PATH java real: %s\n", result.PathJavaResolved)
			}
		} else {
			fmt.Println("PATH java:      (not found)")
		}

		if result.ActiveVersion != "" {
			fmt.Printf("Active version: %s\n", result.ActiveVersion)
		}

		pathOrder := scanner.CheckPATHJavaOrder()
		if len(pathOrder) > 0 {
			fmt.Println("")
			fmt.Println("PATH Order")
			for i, pathJava := range pathOrder {
				marker := " "
				if i == 0 {
					marker = "*"
				}
				fmt.Printf("%s %s\n", marker, pathJava)
			}
		}

		fmt.Println("")
		fmt.Println("Detected JDKs")
		if len(result.Installations) == 0 {
			fmt.Println("  (none found)")
		}
		for _, install := range result.Installations {
			var labels []string
			if install.IsPathActive {
				labels = append(labels, "PATH")
			}
			if install.IsJavaHome {
				labels = append(labels, "JAVA_HOME")
			}
			label := ""
			if len(labels) > 0 {
				label = fmt.Sprintf(" [%s]", strings.Join(labels, ", "))
			}

			fmt.Printf("- %s%s\n", install.Path, label)
			if install.ResolvedPath != "" && install.ResolvedPath != install.Path {
				fmt.Printf("  Real Path: %s\n", install.ResolvedPath)
			}
			if install.Version != "" || install.Vendor != "" {
				fmt.Printf("  Version:   %s\n", install.Version)
				if install.Vendor != "" {
					fmt.Printf("  Vendor:    %s\n", install.Vendor)
				}
			}
			if install.ErrorMsg != "" {
				fmt.Printf("  Error:     %s\n", install.ErrorMsg)
			}
			if install.Source != "" && install.Source != "DISCOVERY" {
				fmt.Printf("  Source:    %s\n", install.Source)
			}
		}

		if len(result.Issues) > 0 {
			fmt.Println("")
			fmt.Println("Issues")
			for _, issue := range result.Issues {
				fmt.Printf("- %s\n", issue)
			}
		} else {
			fmt.Println("")
			fmt.Println("Issues")
			fmt.Println("- No obvious JDK configuration issues detected.")
		}
	},
}

func init() {
	rootCmd.AddCommand(jdkCmd)
}
