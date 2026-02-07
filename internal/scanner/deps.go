package scanner

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type Dependency struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type Project struct {
	Dependencies []Dependency `xml:"dependencies>dependency"`
}

type Conflict struct {
	Artifact string   `json:"artifact"`
	Versions []string `json:"versions"`
}

// GetProjectDependencies returns a list of dependencies from pom.xml or build.gradle
func GetProjectDependencies() ([]Dependency, error) {
	// 1. Try Maven (pom.xml)
	if _, err := os.Stat("pom.xml"); err == nil {
		file, err := os.Open("pom.xml")
		if err != nil {
			return nil, err
		}
		defer file.Close()
		byteValue, _ := io.ReadAll(file)
		var project Project
		if err := xml.Unmarshal(byteValue, &project); err != nil {
			return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
		}
		return project.Dependencies, nil
	}

	// 2. Try Gradle (build.gradle or build.gradle.kts)
	if _, err := os.Stat("build.gradle"); err == nil {
		return parseGradleFile("build.gradle")
	}
	if _, err := os.Stat("build.gradle.kts"); err == nil {
		return parseGradleFile("build.gradle.kts")
	}

	return nil, fmt.Errorf("no supported build file found (pom.xml, build.gradle, build.gradle.kts)")
}

func parseGradleFile(path string) ([]Dependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(file)
	
	// Regex for Gradle dependencies:
	// implementation 'group:artifact:version' or implementation("group:artifact:version")
	// configurations: implementation, api, compileOnly, runtimeOnly, testImplementation, etc.
	// We want to capture the 3 parts.
	// Pattern: (config)\s*\(?['"]([^:]+):([^:]+):([^:]+)['"]\)?
	// Limit to specific configurations to avoid noise? or just match the pattern.
	// Common configs: implementation, api, compile, testImplementation, androidTestImplementation
	
	pattern := regexp.MustCompile(`(?i)(implementation|api|compile|runtime|testImplementation)\s*\(?['"]([^:\s"']+):([^:\s"']+):([^:\s"']+)['"]\)?`)
	
	// TODO: Support map style: implementation group: '...', name: '...', version: '...' (harder with regex)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		
		matches := pattern.FindStringSubmatch(line)
		if len(matches) == 5 {
			// matches[0] is full match
			// matches[1] is config (e.g. implementation)
			// matches[2] is group
			// matches[3] is artifact
			// matches[4] is version
			deps = append(deps, Dependency{
				GroupId:    matches[2],
				ArtifactId: matches[3],
				Version:    matches[4],
			})
		}
	}
	
	return deps, scanner.Err()
}

func ScanDeps() ([]Conflict, error) {
	deps, err := GetProjectDependencies()
	if err != nil {
		return nil, err
	}

	versionMap := make(map[string][]string)
	for _, dep := range deps {
		key := fmt.Sprintf("%s:%s", dep.GroupId, dep.ArtifactId)
		versionMap[key] = append(versionMap[key], dep.Version)
	}

	var conflicts []Conflict
	for artifact, versions := range versionMap {
		if len(versions) > 1 {
			conflicts = append(conflicts, Conflict{
				Artifact: artifact,
				Versions: versions,
			})
		}
	}

	return conflicts, nil
}
