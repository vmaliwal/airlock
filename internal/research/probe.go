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
	detectedFiles := detectFiles(repoRoot)
	repoType := "unknown"
	switch {
	case contains(detectedFiles, "go.mod"):
		repoType = "go"
	case contains(detectedFiles, "package.json"):
		repoType = "node"
	case contains(detectedFiles, "pyproject.toml"):
		repoType = "python"
	case contains(detectedFiles, "Cargo.toml"):
		repoType = "rust"
	}
	return RepoProfile{
		RepoPath:         startPath,
		RepoRoot:         repoRoot,
		TargetPath:       targetPath,
		RepoType:         repoType,
		DetectedFiles:    detectedFiles,
		BaselineCommands: baselineCommandsFor(repoType, repoRoot, targetPath),
	}, nil
}

func AssessRepo(profile RepoProfile) (RepoAssessment, error) {
	blockers := []string{}
	evidence := []string{"repo_root=" + profile.RepoRoot, "target_path=" + profile.TargetPath}
	hostRunnable := true
	vmRunnable := true
	toolchainBlocked := false
	if profile.RepoType == "go" {
		goModBytes, err := os.ReadFile(filepath.Join(profile.RepoRoot, "go.mod"))
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
				abs := filepath.Join(profile.RepoRoot, rel)
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
		hostRunnable = false
		vmRunnable = false
		recommended = "none"
		modes = []string{"structural"}
	} else if toolchainBlocked {
		status = "host_toolchain_blocked_vm_runnable"
		recommended = "vm"
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
	}, nil
}

func detectRepoRoot(start string) string {
	current := start
	for {
		if exists(filepath.Join(current, ".git")) || exists(filepath.Join(current, "go.mod")) || exists(filepath.Join(current, "package.json")) || exists(filepath.Join(current, "pyproject.toml")) || exists(filepath.Join(current, "Cargo.toml")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start
		}
		current = parent
	}
}

func detectFiles(root string) []string {
	files := []string{}
	for _, name := range []string{"package.json", "go.mod", "pyproject.toml", "Cargo.toml"} {
		if exists(filepath.Join(root, name)) {
			files = append(files, name)
		}
	}
	return files
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
func contains(items []string, v string) bool {
	for _, item := range items {
		if item == v {
			return true
		}
	}
	return false
}
