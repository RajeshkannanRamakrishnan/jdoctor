package scanner

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type JDKInstallation struct {
	Path         string `json:"path"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Version      string `json:"version,omitempty"`
	Vendor       string `json:"vendor,omitempty"`
	Source       string `json:"source"`
	IsJavaHome   bool   `json:"is_java_home"`
	IsPathActive bool   `json:"is_path_active"`
	ErrorMsg     string `json:"error,omitempty"`
}

type JDKCheckResult struct {
	JavaHome         string            `json:"java_home,omitempty"`
	JavaHomeResolved string            `json:"java_home_resolved,omitempty"`
	PathJava         string            `json:"path_java,omitempty"`
	PathJavaResolved string            `json:"path_java_resolved,omitempty"`
	ActiveVersion    string            `json:"active_version,omitempty"`
	Installations    []JDKInstallation `json:"installations"`
	Issues           []string          `json:"issues,omitempty"`
}

func ScanJDKs() JDKCheckResult {
	result := JDKCheckResult{}
	seen := make(map[string]int)

	javaHome := strings.TrimSpace(os.Getenv("JAVA_HOME"))
	if javaHome != "" {
		result.JavaHome = javaHome
		resolved := resolvePath(javaHome)
		result.JavaHomeResolved = resolved
		install := inspectJDKHome(javaHome, "JAVA_HOME")
		install.IsJavaHome = true
		addOrMergeJDK(&result.Installations, seen, install)
		if install.ErrorMsg != "" {
			result.Issues = append(result.Issues, fmt.Sprintf("JAVA_HOME points to an invalid JDK: %s", install.ErrorMsg))
		}
	} else {
		result.Issues = append(result.Issues, "JAVA_HOME is not set.")
	}

	if pathJava, err := exec.LookPath("java"); err == nil {
		result.PathJava = pathJava
		result.PathJavaResolved = resolvePath(pathJava)
		activeHome := detectJavaHomeFromBinary(pathJava)
		if activeHome == "" {
			activeHome = guessJavaHomeFromBinary(pathJava)
		}
		install := inspectJDKHome(activeHome, "PATH")
		install.IsPathActive = true
		if install.Path == "" {
			install.Path = activeHome
		}
		if install.ResolvedPath == "" {
			install.ResolvedPath = resolvePath(activeHome)
		}
		addOrMergeJDK(&result.Installations, seen, install)
		result.ActiveVersion = install.Version
		if install.ErrorMsg != "" {
			result.Issues = append(result.Issues, fmt.Sprintf("The java executable found on PATH is not usable: %s", install.ErrorMsg))
		}
	} else {
		result.Issues = append(result.Issues, "No java executable was found on PATH.")
	}

	for _, discovered := range discoverInstalledJDKHomes() {
		install := inspectJDKHome(discovered, "DISCOVERY")
		addOrMergeJDK(&result.Installations, seen, install)
	}

	if result.JavaHomeResolved != "" && result.PathJavaResolved != "" {
		pathHome := detectJavaHomeFromBinary(result.PathJava)
		if pathHome == "" {
			pathHome = guessJavaHomeFromBinary(result.PathJava)
		}
		pathHomeResolved := resolvePath(pathHome)
		if pathHomeResolved != "" && result.JavaHomeResolved != pathHomeResolved {
			result.Issues = append(result.Issues, "JAVA_HOME and the active java on PATH point to different JDK installations.")
		}
	}

	if len(result.Installations) == 0 {
		result.Issues = append(result.Issues, "No JDK installations were discovered.")
	}

	sort.Slice(result.Installations, func(i, j int) bool {
		if result.Installations[i].IsPathActive != result.Installations[j].IsPathActive {
			return result.Installations[i].IsPathActive
		}
		if result.Installations[i].IsJavaHome != result.Installations[j].IsJavaHome {
			return result.Installations[i].IsJavaHome
		}
		return result.Installations[i].Path < result.Installations[j].Path
	})

	return result
}

func addOrMergeJDK(installations *[]JDKInstallation, seen map[string]int, install JDKInstallation) {
	key := install.ResolvedPath
	if key == "" {
		key = install.Path
	}
	if key == "" {
		return
	}

	if idx, ok := seen[key]; ok {
		existing := &(*installations)[idx]
		existing.IsJavaHome = existing.IsJavaHome || install.IsJavaHome
		existing.IsPathActive = existing.IsPathActive || install.IsPathActive
		if existing.Source == "" || existing.Source == "DISCOVERY" {
			existing.Source = install.Source
		}
		if existing.Version == "" {
			existing.Version = install.Version
		}
		if existing.Vendor == "" {
			existing.Vendor = install.Vendor
		}
		if existing.ErrorMsg == "" {
			existing.ErrorMsg = install.ErrorMsg
		}
		return
	}

	seen[key] = len(*installations)
	*installations = append(*installations, install)
}

func inspectJDKHome(home, source string) JDKInstallation {
	install := JDKInstallation{
		Path:         home,
		ResolvedPath: resolvePath(home),
		Source:       source,
	}

	if home == "" {
		install.ErrorMsg = "empty path"
		return install
	}

	javaBin := filepath.Join(home, "bin", javaBinaryName())
	if _, err := os.Stat(javaBin); err != nil {
		install.ErrorMsg = "missing bin/java"
		return install
	}

	version, vendor, resolvedHome, err := readJavaMetadata(javaBin)
	if err != nil {
		install.ErrorMsg = err.Error()
		return install
	}
	install.Version = version
	install.Vendor = vendor
	if resolvedHome != "" {
		install.ResolvedPath = resolvePath(resolvedHome)
		if install.Path == "" {
			install.Path = resolvedHome
		}
	}

	return install
}

func readJavaMetadata(javaBin string) (string, string, string, error) {
	cmd := exec.Command(javaBin, "-XshowSettings:properties", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to inspect java: %w", err)
	}

	version := parseJavaProperty(string(output), "java.version")
	if version == "" {
		version = parseJavaVersion(string(output))
	}
	vendor := parseJavaProperty(string(output), "java.vendor")
	javaHome := parseJavaProperty(string(output), "java.home")
	return version, vendor, javaHome, nil
}

func parseJavaProperty(output, key string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	prefix := key + " = "
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func detectJavaHomeFromBinary(javaBin string) string {
	if javaBin == "" {
		return ""
	}
	_, _, javaHome, err := readJavaMetadata(javaBin)
	if err != nil {
		return ""
	}
	return javaHome
}

func guessJavaHomeFromBinary(javaBin string) string {
	if javaBin == "" {
		return ""
	}

	resolved := resolvePath(javaBin)
	if resolved == "" {
		resolved = javaBin
	}
	dir := filepath.Dir(resolved)
	if strings.EqualFold(filepath.Base(dir), "bin") {
		return filepath.Dir(dir)
	}
	return filepath.Dir(dir)
}

func resolvePath(path string) string {
	if path == "" {
		return ""
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved
	}
	return abs
}

func discoverInstalledJDKHomes() []string {
	seen := make(map[string]bool)
	var homes []string

	addHome := func(path string) {
		if path == "" {
			return
		}
		resolved := resolvePath(path)
		if resolved == "" {
			resolved = path
		}
		if seen[resolved] {
			return
		}
		seen[resolved] = true
		homes = append(homes, path)
	}

	switch runtime.GOOS {
	case "darwin":
		for _, home := range discoverMacJDKHomes() {
			addHome(home)
		}
	case "linux":
		for _, home := range discoverLinuxJDKHomes() {
			addHome(home)
		}
	case "windows":
		for _, home := range discoverWindowsJDKHomes() {
			addHome(home)
		}
	}

	return homes
}

func discoverMacJDKHomes() []string {
	var homes []string

	cmd := exec.Command("/usr/libexec/java_home", "-V")
	output, err := cmd.CombinedOutput()
	if err == nil || len(output) > 0 {
		re := regexp.MustCompile(`(?m)^\s*([0-9][^,]*)[, ].*?(/.*)$`)
		for _, match := range re.FindAllStringSubmatch(string(output), -1) {
			if len(match) == 3 {
				homes = append(homes, strings.TrimSpace(match[2]))
			}
		}
	}

	matches, _ := filepath.Glob("/Library/Java/JavaVirtualMachines/*/Contents/Home")
	homes = append(homes, matches...)
	return homes
}

func discoverLinuxJDKHomes() []string {
	patterns := []string{
		"/usr/lib/jvm/*",
		"/usr/java/*",
		"/opt/java/*",
		"/opt/jdk*",
	}

	var homes []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			if looksLikeJDKHome(match) {
				homes = append(homes, match)
			}
		}
	}
	return homes
}

func discoverWindowsJDKHomes() []string {
	var roots []string
	for _, envName := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if base := os.Getenv(envName); base != "" {
			roots = append(roots, filepath.Join(base, "Java"))
		}
	}

	var homes []string
	for _, root := range roots {
		matches, _ := filepath.Glob(filepath.Join(root, "*"))
		for _, match := range matches {
			if looksLikeJDKHome(match) {
				homes = append(homes, match)
			}
		}
	}
	return homes
}

func looksLikeJDKHome(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(path, "bin", javaBinaryName()))
	return err == nil
}

func javaBinaryName() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

func CheckPATHJavaOrder() []string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil
	}

	var matches []string
	for _, entry := range filepath.SplitList(pathEnv) {
		if entry == "" {
			continue
		}
		javaPath := filepath.Join(entry, javaBinaryName())
		if _, err := os.Stat(javaPath); err == nil {
			matches = append(matches, javaPath)
		}
	}

	var deduped []string
	seen := make(map[string]bool)
	for _, match := range matches {
		resolved := resolvePath(match)
		key := resolved
		if key == "" {
			key = match
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, match)
	}
	return deduped
}

func SummarizePATHJavaOrder() string {
	matches := CheckPATHJavaOrder()
	if len(matches) == 0 {
		return ""
	}

	var buf bytes.Buffer
	for i, match := range matches {
		if i > 0 {
			buf.WriteString(" -> ")
		}
		buf.WriteString(match)
	}
	return buf.String()
}
