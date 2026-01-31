package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "Start a Java REPL (jshell) with project dependencies",
	Long:  `Launches jshell with the runtime classpath of your Maven or Gradle project pre-loaded.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Resolving project classpath...")
		
		cp, err := scanner.GetProjectClasspath()
		if err != nil {
			fmt.Printf("❌ Failed to resolve classpath: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("🚀 Launching jshell...")
		
		// Run jshell
		// We use the 'java' executable from PATH usually, assuming 'jshell' is alongside it.
		// Or just "jshell" if in path.
		jshellCmd := exec.Command("jshell", "--class-path", cp)
		jshellCmd.Stdin = os.Stdin
		jshellCmd.Stdout = os.Stdout
		jshellCmd.Stderr = os.Stderr

		if err := jshellCmd.Run(); err != nil {
			fmt.Printf("❌ jshell exited with error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(replCmd)
}
