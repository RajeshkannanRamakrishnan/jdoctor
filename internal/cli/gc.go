package cli

import (
	"fmt"
	"jdoctor/internal/scanner"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Garbage Collection analysis tools",
}

var monitorCmd = &cobra.Command{
	Use:   "monitor [PID]",
	Short: "Real-time GC monitoring dashboard",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pid := args[0]
		
		// Validate PID exists
		if !scanner.CheckPidExists(pid) {
			fmt.Printf("❌ Process with PID %s not found.\n", pid)
			return
		}

		fmt.Printf("Starting GC monitor for PID %s... (Press Ctrl+C to exit)\n", pid)
		
		// Channels for updates
		updates := make(chan scanner.GCStats)
		errors := make(chan error)
		
		// Start monitoring in background
		go scanner.MonitorGC(pid, updates, errors)
		
		// Setup signal handling for graceful exit
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// TUI Loop
		for {
			select {
			case stats := <-updates:
				renderDashboard(stats, pid)
			case err := <-errors:
				fmt.Printf("\n❌ Error: %v\n", err)
				return
			case <-sigChan:
				fmt.Println("\nStopping monitor...")
				return
			}
		}
	},
}

func renderDashboard(s scanner.GCStats, pid string) {
	// Clear screen and move cursor to top-left
	// \033[H = Move cursor to home (top-left)
	// \033[2J = Clear entire screen
	fmt.Print("\033[H\033[2J")

	fmt.Printf("🔍 Java GC Monitor: PID %s\n", pid)
	fmt.Printf("   Time: %s\n", time.Now().Format("15:04:05"))
	fmt.Println(strings.Repeat("-", 50))
	
	fmt.Println("\n🧠 Heap Usage:")
	printBar("Survivor 0", s.S0)
	printBar("Survivor 1", s.S1)
	printBar("Eden Space", s.Eden)
	printBar("Old Gen   ", s.Old)
	printBar("Metaspace ", s.Meta)

	fmt.Println("\n♻️  GC Activity:")
	fmt.Printf("   Young GC: %-5d events (Total Time: %.3fs)\n", s.YGC, s.YGCT)
	fmt.Printf("   Full GC:  %-5d events (Total Time: %.3fs)\n", s.FGC, s.FGCT)
	fmt.Printf("   Total GC Time: %.3fs\n", s.GCT)
	
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("Press Ctrl+C to exit")
}

func printBar(label string, percent float64) {
	barWidth := 20
	// Clamp percent between 0 and 100
	if percent < 0 { percent = 0 }
	if percent > 100 { percent = 100 }

	filledLen := int((percent / 100.0) * float64(barWidth))
	emptyLen := barWidth - filledLen
	
	bar := strings.Repeat("█", filledLen) + strings.Repeat("░", emptyLen)
	
	color := "\033[32m" // Green
	if percent > 70 {
		color = "\033[33m" // Yellow
	} 
	if percent > 90 {
		color = "\033[31m" // Red
	}
	reset := "\033[0m"

	fmt.Printf("   %-12s [%s%s%s] %6.2f%%\n", label, color, bar, reset, percent)
}

func init() {
	gcCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(gcCmd)
}
