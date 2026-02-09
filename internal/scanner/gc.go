package scanner

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GCStats represents the parsed output of `jstat -gcutil`
type GCStats struct {
	S0   float64 // Survivor space 0 utilization
	S1   float64 // Survivor space 1 utilization
	Eden float64 // Eden space utilization
	Old  float64 // Old space utilization
	Meta float64 // Metaspace utilization
	YGC  int64   // Number of young generation GC events
	YGCT float64 // Young generation garbage collection time
	FGC  int64   // Number of full GC events
	FGCT float64 // Full garbage collection time
	GCT  float64 // Total garbage collection time
}

// MonitorGC runs jstat iteratively and sends updates to the provided channel
func MonitorGC(pid string, updates chan<- GCStats, errors chan<- error) {
	// check if jstat exists
	_, err := exec.LookPath("jstat")
	if err != nil {
		errors <- fmt.Errorf("jstat not found in PATH. Please ensure JDK is installed")
		return
	}

	// Run jstat -gcutil <pid> 1000ms
	cmd := exec.Command("jstat", "-gcutil", pid, "1000")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errors <- err
		return
	}

	if err := cmd.Start(); err != nil {
		errors <- fmt.Errorf("failed to start jstat: %w", err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	headerParsed := false
	
	// Column indices map
	cols := make(map[string]int)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)

		// Parse Header
		if !headerParsed {
			// Expected columns: S0 S1 E O M ...
			for i, col := range parts {
				cols[col] = i
			}
			// Verify critical columns exist
			if _, ok := cols["E"]; !ok {
				errors <- fmt.Errorf("unexpected jstat output format: missing 'E' column")
				return
			}
			headerParsed = true
			continue
		}

		// Parse Data
		stats := GCStats{}
		
		// Helper to safely parse float
		parseFloat := func(name string) float64 {
			if idx, ok := cols[name]; ok && idx < len(parts) {
				val, _ := strconv.ParseFloat(parts[idx], 64)
				return val
			}
			return 0.0
		}
		
		// Helper to safely parse int
		parseInt := func(name string) int64 {
			if idx, ok := cols[name]; ok && idx < len(parts) {
				val, _ := strconv.ParseInt(parts[idx], 10, 64)
				return val
			}
			return 0
		}

		stats.S0 = parseFloat("S0")
		stats.S1 = parseFloat("S1")
		stats.Eden = parseFloat("E")
		stats.Old = parseFloat("O")
		stats.Meta = parseFloat("M") // might be "M" or "Mu" depending on version? usually M in gcutil
		if _, ok := cols["M"]; !ok && stats.Meta == 0 {
             stats.Meta = parseFloat("MC") // Fallback if regular M missing? typically gcutil returns M
        }

		stats.YGC = parseInt("YGC")
		stats.YGCT = parseFloat("YGCT")
		stats.FGC = parseInt("FGC")
		stats.FGCT = parseFloat("FGCT")
		stats.GCT = parseFloat("GCT")

		updates <- stats
	}

	if err := cmd.Wait(); err != nil {
		// If process exits, jstat exits
		errors <- fmt.Errorf("process terminated or jstat exited: %v", err)
	}
}

// CheckPidExists verifies if a PID exists (simplistic check)
func CheckPidExists(pid string) bool {
	// On unix, kill -0 <pid> is a good check
	// But let's just check /proc or run ps
	// `os.FindProcess` always returns a process on Unix, need `Signal` to check.
	// Easiest cross-platform way for jdoctor might be trying `jps` again or `ps -p`
	cmd := exec.Command("ps", "-p", pid)
	return cmd.Run() == nil
}
