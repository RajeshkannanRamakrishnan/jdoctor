package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	cacheFileName = "vuln_cache.json"
	cacheTTL      = 24 * time.Hour
)

type CacheEntry struct {
	Timestamp time.Time    `json:"timestamp"`
	Result    []Vulnerability `json:"result"`
}

type VulnCache struct {
	Entries map[string]CacheEntry `json:"entries"`
	mu      sync.RWMutex
	path    string
}

func NewVulnCache() (*VulnCache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	cacheDir := filepath.Join(home, ".jdoctor")
	return &VulnCache{
		Entries: make(map[string]CacheEntry),
		path:    filepath.Join(cacheDir, cacheFileName),
	}, nil
}

func (c *VulnCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return nil // New cache
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &c.Entries)
}

func (c *VulnCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c.Entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// Get returns cached vulnerabilities if they exist and are not expired
func (c *VulnCache) Get(packageURL string) ([]Vulnerability, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.Entries[packageURL]
	if !found {
		return nil, false
	}

	if time.Since(entry.Timestamp) > cacheTTL {
		return nil, false
	}

	return entry.Result, true
}

func (c *VulnCache) Set(packageURL string, vulns []Vulnerability) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Entries[packageURL] = CacheEntry{
		Timestamp: time.Now(),
		Result:    vulns,
	}
}
