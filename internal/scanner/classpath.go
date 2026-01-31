package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetProjectClasspath attempts to detect the build tool (Maven/Gradle)
// and returns the project's runtime classpath as a string string.
func GetProjectClasspath() (string, error) {
	if isMaven() {
		return getMavenClasspath()
	}
	if isGradle() {
		return getGradleClasspath()
	}
	return "", fmt.Errorf("no supported build tool found (pom.xml or build.gradle)")
}

func isMaven() bool {
	_, err := os.Stat("pom.xml")
	return err == nil
}

func isGradle() bool {
	if _, err := os.Stat("build.gradle"); err == nil {
		return true
	}
	if _, err := os.Stat("build.gradle.kts"); err == nil {
		return true
	}
	return false
}

func getMavenClasspath() (string, error) {
	// Use mvn dependency:build-classpath
	// We'll write to a temp file to avoid parsing noise from stdout
	tmpFile, err := os.CreateTemp("", "cp-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // close immediately, maven will write to it
	defer os.Remove(tmpPath)

	cmdName := "mvn"
	if _, err := os.Stat("mvnw"); err == nil {
		cmdName = "./mvnw"
	}

	cmd := exec.Command(cmdName, "dependency:build-classpath", "-Dmdep.outputFile="+tmpPath, "-q")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("maven failed: %s, output: %s", err, string(output))
	}

	cpBytes, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read classpath file: %w", err)
	}

	return strings.TrimSpace(string(cpBytes)), nil
}

func getGradleClasspath() (string, error) {
	// Create a temporary init script to safely extract classpath
	initScript := `
allprojects {
    task jdoctorPrintClasspath {
        doLast {
            println sourceSets.main.runtimeClasspath.asPath
        }
    }
}
`
	tmpInit, err := os.CreateTemp("", "init-*.gradle")
	if err != nil {
		return "", fmt.Errorf("failed to create temp init script: %w", err)
	}
	defer os.Remove(tmpInit.Name())

	if _, err := tmpInit.WriteString(initScript); err != nil {
		return "", fmt.Errorf("failed to write init script: %w", err)
	}
	tmpInit.Close()

	cmdName := "gradle"
	if _, err := os.Stat("gradlew"); err == nil {
		cmdName = "./gradlew"
	}

	// Run the injected task
	cmd := exec.Command(cmdName, "--init-script", tmpInit.Name(), "jdoctorPrintClasspath", "-q", "--console=plain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gradle failed: %s, output: %s", err, string(output))
	}

	// Output might contain other noise, but with -q it should be clean. 
	// However, sometimes gradle prints "Welcome to Gradle..." etc. 
	// We'll take the last non-empty line ideally, or just the whole output if it's clean.
	// The init script prints ONLY the path in the task action.
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("gradle produced no output")
	}
	// Return the last line as it's most likely the printed classpath
	return strings.TrimSpace(lines[len(lines)-1]), nil
}
