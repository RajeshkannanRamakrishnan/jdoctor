package scanner

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type EnvInfo struct {
	JavaVersion string `json:"java_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Error       error  `json:"-"`
	ErrorMsg    string `json:"error,omitempty"`
}

func ScanEnv() EnvInfo {
	info := EnvInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	cmd := exec.Command("java", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		info.Error = fmt.Errorf("java not found or not executable: %w", err)
		info.ErrorMsg = info.Error.Error()
		return info
	}

	// Parse version from output (e.g., "openjdk version 17.0.1 ...")
	info.JavaVersion = parseJavaVersion(string(output))
	return info
}

func parseJavaVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "version") {
			// Very basic parser, can be improved
			parts := strings.Split(line, "\"")
			if len(parts) > 1 {
				return parts[1]
			}
		}
	}
	return "unknown"
}
