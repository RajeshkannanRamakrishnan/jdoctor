package scanner

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
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

func ScanDeps() ([]Conflict, error) {
	// MVP: Only checks pom.xml in current directory
	file, err := os.Open("pom.xml")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("pom.xml not found")
		}
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var project Project
	if err := xml.Unmarshal(byteValue, &project); err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	versionMap := make(map[string][]string)
	for _, dep := range project.Dependencies {
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
