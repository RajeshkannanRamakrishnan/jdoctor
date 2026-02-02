package scanner

import (
	"fmt"
	"os/exec"
	"strings"
)

type JavaProcess struct {
	PID         string
	Name        string
	Args        []string
	Uptime      string
}

func ScanJavaProcesses() ([]JavaProcess, error) {
	// 1. Run jps -v to get PID, Name, and JVM Args
	cmd := exec.Command("jps", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run jps: %w. Ensure JDK is installed and in PATH", err)
	}

	lines := strings.Split(string(output), "\n")
	var processes []JavaProcess

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue // Skip malformed lines
		}

		pid := parts[0]
		remaining := parts[1]

		// Split Name and Args
		// jps output usually looks like: PID Name [Args...]
		// But Name can be a full class path or Jar path.
		// If args start with -, the first part is the name.
		// This is a bit tricky, relying on space separation can be flaky but standard for jps.

		var name string
		var argsStr string
		
		spaceIdx := strings.Index(remaining, " ")
		if spaceIdx == -1 {
			name = remaining
			argsStr = ""
		} else {
			name = remaining[:spaceIdx]
			argsStr = remaining[spaceIdx+1:]
		}
		
		// If name is "Jps", skip it (don't list the scanner itself ideally, or keep it?)
		// Let's keep it for accuracy, but user might filter it.

		args := strings.Fields(argsStr)
		
		// 2. Get Uptime for this PID
		uptime := getUptime(pid)

		processes = append(processes, JavaProcess{
			PID:    pid,
			Name:   name,
			Args:   args,
			Uptime: uptime,
		})
	}

	return processes, nil
}

func getUptime(pid string) string {
	// Attempt to use ps command to get uptime
	// MacOS/Linux: ps -p <pid> -o etime=
	cmd := exec.Command("ps", "-p", pid, "-o", "etime=")
	out, err := cmd.Output()
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(out))
}
