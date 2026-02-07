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

	// Initialize cache
	cache, err := NewVulnCache()
	if err == nil {
		_ = cache.Load()
	}

	var results []ScanResult
	var uncachedDeps []Dependency

	// Check cache
	for _, d := range deps {
		// Use PURL-like key: pkg:maven/group/artifact@version
		key := fmt.Sprintf("pkg:maven/%s/%s@%s", d.GroupId, d.ArtifactId, d.Version)
		
		if cache != nil {
			if vulns, hit := cache.Get(key); hit {
				// Cache hit
				if len(vulns) > 0 {
					results = append(results, ScanResult{
						Dependency: d,
						Vulns:      vulns,
					})
				}
				continue
			}
		}
		
		// Cache miss
		uncachedDeps = append(uncachedDeps, d)
	}

	if len(uncachedDeps) == 0 {
		return results, nil // All served from cache
	}

	// Query OSV for uncached dependencies
	reqBody := OSVRequest{Queries: make([]OSVQuery, len(uncachedDeps))}
	for i, d := range uncachedDeps {
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

	// Process results and update cache
	for i, res := range osvResp.Results {
		originalDep := uncachedDeps[i]
		key := fmt.Sprintf("pkg:maven/%s/%s@%s", originalDep.GroupId, originalDep.ArtifactId, originalDep.Version)

		// Update cache (store even if empty to avoid re-querying safe packages)
		if cache != nil {
			cache.Set(key, res.Vulns)
		}

		if len(res.Vulns) > 0 {
			results = append(results, ScanResult{
				Dependency: originalDep,
				Vulns:      res.Vulns,
			})
		}
	}

	// Save cache
	if cache != nil {
		_ = cache.Save()
	}

	return results, nil
}
