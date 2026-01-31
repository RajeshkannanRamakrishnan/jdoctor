package scanner

import (
	"fmt"
	"os"
	"os/exec"
)

type BuildToolInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Healthy  bool   `json:"healthy"`
	Error    error  `json:"-"`
	ErrorMsg string `json:"error,omitempty"`
}

func ScanBuild() []BuildToolInfo {
	var results []BuildToolInfo

	// Check Maven
	if _, err := os.Stat("mvnw"); err == nil {
		results = append(results, checkWrapper("Maven", "./mvnw"))
	} else if _, err := os.Stat("pom.xml"); err == nil {
		results = append(results, checkSystemTool("Maven", "mvn"))
	}

	// Check Gradle
	if _, err := os.Stat("gradlew"); err == nil {
		results = append(results, checkWrapper("Gradle", "./gradlew"))
	} else if _, err := os.Stat("build.gradle"); err == nil || fileExists("build.gradle.kts") {
		results = append(results, checkSystemTool("Gradle", "gradle"))
	}

	if len(results) == 0 {
		results = append(results, BuildToolInfo{
			Name:     "Unknown",
			Error:    fmt.Errorf("no build tool detected (pom.xml/build.gradle not found)"),
			ErrorMsg: "no build tool detected (pom.xml/build.gradle not found)",
		})
	}

	return results
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func checkWrapper(name, path string) BuildToolInfo {
	cmd := exec.Command(path, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return BuildToolInfo{
			Name:     name,
			Healthy:  false,
			Error:    fmt.Errorf("wrapper execution failed: %w", err),
			ErrorMsg: err.Error(),
		}
	}
	return BuildToolInfo{
		Name:    name,
		Healthy: true,
		Version: string(output), // simplistic, usually contains version
	}
}

func checkSystemTool(name, bin string) BuildToolInfo {
	cmd := exec.Command(bin, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return BuildToolInfo{
			Name:     name,
			Healthy:  false,
			Error:    fmt.Errorf("system tool not found or failed: %w", err),
			ErrorMsg: err.Error(),
		}
	}
	return BuildToolInfo{
		Name:    name,
		Healthy: true,
		Version: string(output),
	}
}
