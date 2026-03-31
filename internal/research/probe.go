package research

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vmaliwal/airlock/internal/util"
)

func DetectRepo(startPath string) (RepoProfile, error) {
	targetPath, err := filepath.Abs(startPath)
	if err != nil {
		return RepoProfile{}, err
	}
	repoRoot := detectRepoRoot(targetPath)
	scopeRoot := detectScopeRoot(targetPath, repoRoot)
	detectedFiles := detectFiles(scopeRoot)
	discoveredTargets := []string{}
	bootstrapHints := []string{}
	repoType := "unknown"
	switch {
	case contains(detectedFiles, "go.mod"):
		repoType = "go"
	case contains(detectedFiles, "package.json"):
		repoType = "node"
		bootstrapHints = nodeBootstrapHints(scopeRoot)
	case contains(detectedFiles, "pyproject.toml"):
		repoType = "python"
		bootstrapHints = pythonBootstrapHints(scopeRoot)
	case contains(detectedFiles, "Cargo.toml"):
		repoType = "rust"
	default:
		discoveredTargets = discoverNestedTargets(repoRoot, 3)
	}
	return RepoProfile{
		RepoPath:          startPath,
		RepoRoot:          repoRoot,
		ScopeRoot:         scopeRoot,
		TargetPath:        targetPath,
		RepoType:          repoType,
		DetectedFiles:     detectedFiles,
		DiscoveredTargets: discoveredTargets,
		BaselineCommands:  baselineCommandsFor(repoType, scopeRoot, targetPath),
		BootstrapHints:    bootstrapHints,
	}, nil
}

func AssessRepo(profile RepoProfile) (RepoAssessment, error) {
	blockers := []string{}
	warnings := []string{}
	evidence := []string{"repo_root=" + profile.RepoRoot, "target_path=" + profile.TargetPath}
	if profile.ScopeRoot != "" && !samePath(profile.ScopeRoot, profile.RepoRoot) {
		evidence = append(evidence, "scope_root="+profile.ScopeRoot)
	}
	hostRunnable := true
	vmRunnable := true
	toolchainBlocked := false
	if profile.RepoType == "unknown" && len(profile.DiscoveredTargets) > 0 && samePath(profile.TargetPath, profile.RepoRoot) {
		blockers = append(blockers, "monorepo root needs a concrete package target")
		evidence = append(evidence, "discovered_nested_targets="+strconv.Itoa(len(profile.DiscoveredTargets)))
		for _, target := range profile.DiscoveredTargets {
			evidence = append(evidence, "candidate_target="+target)
		}
	}
	if profile.RepoType == "go" {
		goRoot := profile.ScopeRoot
		if goRoot == "" {
			goRoot = profile.RepoRoot
		}
		goModBytes, err := os.ReadFile(filepath.Join(goRoot, "go.mod"))
		if err == nil {
			goMod := string(goModBytes)
			if req := detectGoDirective(goMod); req != "" {
				evidence = append(evidence, "go_mod_version="+req)
				if local := detectLocalGoVersion(); local != "" {
					evidence = append(evidence, "local_go_version="+local)
					if compareGoVersions(local, req) < 0 {
						evidence = append(evidence, "local_go_too_old_for_host_execution")
						hostRunnable = false
						toolchainBlocked = true
					}
				}
			}
			re := regexp.MustCompile(`replace\s+\S+\s+=>\s+(\.\/[^\s]+)`)
			matches := re.FindAllStringSubmatch(goMod, -1)
			evidence = append(evidence, "detected "+strconv.Itoa(len(matches))+" local replace directives in go.mod")
			for _, m := range matches {
				rel := m[1]
				abs := filepath.Join(goRoot, rel)
				info, err := os.Stat(abs)
				if err != nil {
					blockers = append(blockers, "missing local replace target: "+rel)
					continue
				}
				if info.IsDir() {
					entries, err := os.ReadDir(abs)
					if err != nil {
						blockers = append(blockers, "unreadable local replace target: "+rel)
						continue
					}
					if len(entries) == 0 {
						blockers = append(blockers, "empty local replace target: "+rel)
					}
				}
			}
		}
	}
	status := "ready"
	recommended := "host"
	modes := []string{"structural", "functional", "stability", "campaign"}
	if len(blockers) > 0 {
		status = "structurally_blocked"
		if len(profile.DiscoveredTargets) > 0 && samePath(profile.TargetPath, profile.RepoRoot) {
			status = "monorepo_target_required"
		}
		hostRunnable = false
		vmRunnable = false
		recommended = "none"
		modes = []string{"structural"}
	} else if toolchainBlocked {
		status = "host_toolchain_blocked_vm_runnable"
		recommended = "vm"
	} else if len(profile.BootstrapHints) > 0 {
		status = "bootstrap_needed_vm_preferred"
		recommended = "vm"
		evidence = append(evidence, profile.BootstrapHints...)
	} else if profile.RepoType == "unknown" && len(profile.DiscoveredTargets) == 0 {
		status = "env_config_blocked"
		recommended = "vm"
		hostRunnable = false
		evidence = append(evidence, "unknown repo type; concrete runtime/bootstrap context still missing")
	} else if !samePath(profile.TargetPath, profile.RepoRoot) {
		status = "partial_runnable_scope"
		recommended = "vm"
		evidence = append(evidence, "subdir-targeted scope detected")
	}
	if hasServiceHints(profile.DetectedFiles) {
		warnings = append(warnings, "service_dependent")
	}
	if hasIntegrationHints(profile.DetectedFiles) {
		warnings = append(warnings, "integration_blocked")
	}
	return RepoAssessment{
		Runnable:             hostRunnable,
		HostRunnable:         hostRunnable,
		VMRunnable:           vmRunnable,
		RecommendedExecution: recommended,
		Status:               status,
		PossibleModes:        modes,
		Blockers:             blockers,
		Evidence:             evidence,
		Warnings:             dedupeStrings(warnings),
	}, nil
}

func detectRepoRoot(start string) string {
	current := start
	for {
		if exists(filepath.Join(current, ".git")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	current = start
	for {
		if exists(filepath.Join(current, "go.mod")) || exists(filepath.Join(current, "package.json")) || exists(filepath.Join(current, "pyproject.toml")) || exists(filepath.Join(current, "Cargo.toml")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start
		}
		current = parent
	}
}

func detectScopeRoot(targetPath, repoRoot string) string {
	current := targetPath
	for {
		if hasManifest(current) {
			return current
		}
		if samePath(current, repoRoot) {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return repoRoot
}

func hasManifest(root string) bool {
	for _, name := range []string{"package.json", "go.mod", "pyproject.toml", "Cargo.toml"} {
		if exists(filepath.Join(root, name)) {
			return true
		}
	}
	return false
}

func detectFiles(root string) []string {
	files := []string{}
	for _, name := range []string{"package.json", "go.mod", "pyproject.toml", "Cargo.toml", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "requirements.txt", "poetry.lock", "uv.lock", "tox.ini", "docker-compose.yml", "compose.yml"} {
		if exists(filepath.Join(root, name)) {
			files = append(files, name)
		}
	}
	return files
}

func discoverNestedTargets(root string, maxDepth int) []string {
	candidates := []string{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !contains([]string{"package.json", "go.mod", "pyproject.toml", "Cargo.toml"}, info.Name()) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if depth > maxDepth {
			return nil
		}
		dir := filepath.Dir(path)
		if dir == root {
			return nil
		}
		candidates = append(candidates, dir)
		return nil
	})
	return dedupeSorted(candidates)
}

func baselineCommandsFor(repoType, repoRoot, targetPath string) []string {
	rel := strings.TrimPrefix(targetPath, repoRoot)
	rel = strings.TrimPrefix(rel, string(filepath.Separator))
	switch repoType {
	case "go":
		if rel == "" || rel == "." {
			return []string{"go test ./..."}
		}
		return []string{"go test ./" + rel, "go test ./" + rel + "/..."}
	case "node":
		return []string{"npm test"}
	case "python":
		return []string{"pytest"}
	case "rust":
		return []string{"cargo test"}
	default:
		return nil
	}
}

func detectGoDirective(goMod string) string {
	re := regexp.MustCompile(`(?m)^go\s+([0-9]+\.[0-9]+(?:\.[0-9]+)?)$`)
	m := re.FindStringSubmatch(goMod)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func detectLocalGoVersion() string {
	out, err := util.RunLocal("go", []string{"version"}, util.RunOptions{})
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`go([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	m := re.FindStringSubmatch(string(out))
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func compareGoVersions(a, b string) int {
	ap := parseVersionParts(a)
	bp := parseVersionParts(b)
	for len(ap) < len(bp) {
		ap = append(ap, 0)
	}
	for len(bp) < len(ap) {
		bp = append(bp, 0)
	}
	for i := range ap {
		if ap[i] < bp[i] {
			return -1
		}
		if ap[i] > bp[i] {
			return 1
		}
	}
	return 0
}

func parseVersionParts(v string) []int {
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }
func samePath(a, b string) bool {
	aa, err := filepath.Abs(a)
	if err != nil {
		return a == b
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		return a == b
	}
	return aa == bb
}
func dedupeSorted(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
func contains(items []string, v string) bool {
	for _, item := range items {
		if item == v {
			return true
		}
	}
	return false
}

func nodeBootstrapHints(root string) []string {
	hints := []string{}
	if !exists(filepath.Join(root, "package-lock.json")) && !exists(filepath.Join(root, "yarn.lock")) && !exists(filepath.Join(root, "pnpm-lock.yaml")) {
		hints = append(hints, "node repo has no lockfile; bootstrap/install policy likely required before honest execution")
	}
	return hints
}

func pythonBootstrapHints(root string) []string {
	hints := []string{}
	if exists(filepath.Join(root, "pyproject.toml")) {
		hints = append(hints, "python repo likely needs venv-first bootstrap")
	}
	if !exists(filepath.Join(root, "requirements.txt")) && !exists(filepath.Join(root, "poetry.lock")) && !exists(filepath.Join(root, "uv.lock")) {
		hints = append(hints, "python dependency manifest may require project-specific bootstrap")
	}
	return hints
}

func hasServiceHints(files []string) bool {
	for _, name := range []string{"docker-compose.yml", "compose.yml"} {
		if contains(files, name) {
			return true
		}
	}
	return false
}

func hasIntegrationHints(files []string) bool {
	for _, name := range []string{"tox.ini", "docker-compose.yml", "compose.yml"} {
		if contains(files, name) {
			return true
		}
	}
	return false
}

// IssueBodyServiceDependent returns true when the issue body describes a
// runtime that requires a live service (HMR, dev server, websocket, etc.).
// These issues are usually not amenable to bounded offline test reproduction.
func IssueBodyServiceDependent(body string) bool {
	lower := strings.ToLower(body)
	signals := []string{
		"dev server", "devserver", "hmr", "hot reload", "hot module",
		"program reload", "full reload", "websocket", "ws://", "wss://",
		"vite server", "webpack-dev-server", "live reload",
		"open a browser", "open browser", "click", "manually reproduce",
		"stackblitz", "codesandbox", "reproduction link", "repro link",
		"start the server", "run the server", "docker run", "docker-compose up",
	}
	for _, s := range signals {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// IsSetupCommand returns true when a command looks like it is preparing the
// environment rather than asserting failure (e.g. build/install commands that
// precede the actual test assertion).
func IsSetupCommand(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	setupPrefixes := []string{
		"maturin develop", "maturin build",
		"pip install", "uv pip install", "npm install", "pnpm install", "yarn install",
		"go build", "cargo build",
		"make build", "make install",
		"docker build", "docker-compose build",
	}
	for _, prefix := range setupPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
