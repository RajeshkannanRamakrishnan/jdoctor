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
	// 1. Run jps -lv to get PID, Long Name (Path), and JVM Args
	cmd := exec.Command("jps", "-lv")
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

		// PID LongName [Args...]
		// If LongName contains spaces (e.g. path with spaces), jps might be tricky.
		// But usually jps outputs: PID MainClass Args
		// or PID /path/to/jar Args
		
		// Heuristic: The start of args is usually marked by -D or -X or start of options.
		// But standard apps might use args without -. 
		// Simpler heuristic: The second part is the Name/Path until the first space 
		// UNLESS the path itself has spaces? jps usually handles paths with spaces by quoting?
		// Actually jps output is space delimited. If path has spaces, it might break.
		// Let's assume standard space separation for now.
		
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

		args := strings.Fields(argsStr)
		
		// 2. Get Uptime
		uptime := getUptime(pid)

		processes = append(processes, JavaProcess{
			PID:    pid,
			Name:   name, // This is now the Long Name (Full Path or Package)
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
