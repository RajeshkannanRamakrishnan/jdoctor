package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SASTVulnerability struct {
	ID          string // e.g., "HARDCODED_SECRET"
	Description string
	Severity    string // CRITICAL, HIGH, MEDIUM, LOW
	File        string
	Line        int
	Match       string
}

type SASTRule struct {
	ID          string
	Pattern     *regexp.Regexp
	Description string
	Severity    string
}

// Define rules
var sastRules = []SASTRule{
	{
		ID:          "Did you mean to hardcode this secret?",
		Pattern:     regexp.MustCompile(`(?i)(api_key|secret|password|passwd|token)\s*=\s*['"][a-zA-Z0-9_\-]{8,}['"]`),
		Description: "Possible hardcoded secret detected. Use environment variables instead.",
		Severity:    "HIGH",
	},
	{
		ID:          "SQL Injection Risk",
		Pattern:     regexp.MustCompile(`(executeQuery|executeUpdate)\s*\(\s*".*"\s*\+`),
		Description: "Potential SQL Injection via string concatenation. Use PreparedStatement.",
		Severity:    "HIGH",
	},
	{
		ID:          "Command Injection Risk",
		Pattern:     regexp.MustCompile(`Runtime\.getRuntime\(\)\.exec\(\s*[^"]`),
		Description: "Potential Command Injection with dynamic arguments. Validate input carefully.",
		Severity:    "HIGH",
	},
	{
		ID:          "Weak Cryptography (MD5/SHA-1)",
		Pattern:     regexp.MustCompile(`MessageDigest\.getInstance\(\s*"(MD5|SHA-1)"\s*\)`),
		Description: "Weak hashing algorithm detected. Use SHA-256 or stronger.",
		Severity:    "MEDIUM",
	},
	{
		ID:          "Cloud Credential Leaked",
		Pattern:     regexp.MustCompile(`(AWS_ACCESS_KEY_ID|AWS_SECRET_ACCESS_KEY|GOOGLE_API_KEY)`),
		Description: "Potential cloud provider credential token found.",
		Severity:    "CRITICAL",
	},
}

func ScanSourceCode(rootPath string) ([]SASTVulnerability, error) {
	var results []SASTVulnerability

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and hidden files/dirs
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan .java files for now
		if !strings.HasSuffix(info.Name(), ".java") {
			return nil
		}

		vulns, err := scanFile(path)
		if err != nil {
			// Log error but continue scanning other files?
			// fmt.Printf("Error checking file %s: %v\n", path, err)
			return nil 
		}
		results = append(results, vulns...)
		return nil
	})

	return results, err
}

func scanFile(path string) ([]SASTVulnerability, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var vulns []SASTVulnerability
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		
		// Skip comments (simplistic check)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		for _, rule := range sastRules {
			if rule.Pattern.MatchString(line) {
				vulns = append(vulns, SASTVulnerability{
					ID:          rule.ID,
					Description: rule.Description,
					Severity:    rule.Severity,
					File:        path,
					Line:        lineNumber,
					Match:       strings.TrimSpace(line),
				})
			}
		}
	}

	return vulns, scanner.Err()
}
