package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const osvQueryBatchURL = "https://api.osv.dev/v1/querybatch"

type OSVRequest struct {
	Queries []OSVQuery `json:"queries"`
}

type OSVQuery struct {
	Package OSVPackage `json:"package"`
	Version string     `json:"version"`
}

type OSVPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type OSVResponse struct {
	Results []OSVResult `json:"results"`
}

type OSVResult struct {
	Vulns []Vulnerability `json:"vulns"`
}

type Vulnerability struct {
	ID       string    `json:"id"`
	Summary  string    `json:"summary"`
	Details  string    `json:"details"`
	Severity []Severity `json:"severity"`
	References []Reference `json:"references"`
}

type Severity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type Reference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type ScanResult struct {
	Dependency Dependency
	Vulns      []Vulnerability
}

// ScanVulnerabilities takes a list of dependencies and queries OSV for vulnerabilities
func ScanVulnerabilities(deps []Dependency) ([]ScanResult, error) {
	if len(deps) == 0 {
		return nil, nil
	}

	// Prepare batch request
	reqBody := OSVRequest{Queries: make([]OSVQuery, len(deps))}
	for i, d := range deps {
		reqBody.Queries[i] = OSVQuery{
			Package: OSVPackage{
				Name:      fmt.Sprintf("%s:%s", d.GroupId, d.ArtifactId),
				Ecosystem: "Maven",
			},
			Version: d.Version,
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OSV request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(osvQueryBatchURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("OSV API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV API returned status: %s", resp.Status)
	}

	var osvResp OSVResponse
	if err := json.NewDecoder(resp.Body).Decode(&osvResp); err != nil {
		return nil, fmt.Errorf("failed to decode OSV response: %w", err)
	}

	// Map results back to dependencies
	var results []ScanResult
	for i, res := range osvResp.Results {
		if len(res.Vulns) > 0 {
			results = append(results, ScanResult{
				Dependency: deps[i],
				Vulns:      res.Vulns,
			})
		}
	}

	return results, nil
}
