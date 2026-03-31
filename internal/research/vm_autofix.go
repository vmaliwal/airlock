package research

import (
	base64 "encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	base "github.com/vmaliwal/airlock/internal/contract"
)

func NormalizeCloneURL(raw string) string {
	if strings.HasPrefix(raw, "git@") {
		rest := strings.TrimPrefix(raw, "git@")
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			return "https://" + parts[0] + "/" + parts[1]
		}
	}
	return raw
}

func CompileAutofixPlanToVMContract(plan AutofixPlan, backendKind base.BackendKind) (base.Contract, error) {
	gitRoot, err := GitTopLevel(plan.Repo)
	if err != nil {
		return base.Contract{}, fmt.Errorf("detect git root: %w", err)
	}

	// Prefer explicit CloneURL from the plan (set by airlock fix from the issue
	// context). Fall back to reading git remote only for standalone autofix-run.
	cloneURL := strings.TrimSpace(plan.CloneURL)
	if cloneURL == "" {
		raw, err := GitRemoteOrigin(plan.Repo)
		if err != nil {
			return base.Contract{}, fmt.Errorf("detect origin remote: %w", err)
		}
		cloneURL = NormalizeCloneURL(raw)
	}

	headSHA, err := GitHeadSHA(gitRoot)
	if err != nil {
		return base.Contract{}, fmt.Errorf("detect head sha: %w", err)
	}

	// Resolve symlinks before computing relative subdir. On macOS /var/folders
	// is a symlink to /private/var/folders, so git rev-parse --show-toplevel
	// and os.MkdirTemp may return paths that share no common prefix without
	// resolution, producing a wrong relative traversal path.
	resolvedRoot, rootErr := resolveSymlinks(gitRoot)
	resolvedRepo, repoErr := resolveSymlinks(plan.Repo)
	if rootErr != nil {
		resolvedRoot = gitRoot
	}
	if repoErr != nil {
		resolvedRepo = plan.Repo
	}
	relSubdir, err := filepath.Rel(resolvedRoot, resolvedRepo)
	if err != nil {
		return base.Contract{}, fmt.Errorf("resolve subdir: %w", err)
	}
	if relSubdir == "." || strings.HasPrefix(relSubdir, "..") {
		relSubdir = ""
	}
	guestPlan := plan
	guestRepoPath := "/airlock/work/repo"
	if relSubdir != "" {
		guestRepoPath = filepath.ToSlash(filepath.Join(guestRepoPath, relSubdir))
	}
	guestPlan.Repo = guestRepoPath
	guestPlan.ArtifactsDir = "/airlock/artifacts/autofix"
	guestPlan.Checkpoint = ""
	for i := range guestPlan.Attempts {
		guestPlan.Attempts[i].Repo = ""
		guestPlan.Attempts[i].ArtifactsDir = ""
		guestPlan.Attempts[i].Checkpoint = ""
		if relSubdir != "" {
			guestPlan.Attempts[i].Attempt.Safety.AllowedPaths = prefixPaths(relSubdir, guestPlan.Attempts[i].Attempt.Safety.AllowedPaths)
			guestPlan.Attempts[i].Attempt.Safety.ForbiddenPaths = prefixPaths(relSubdir, guestPlan.Attempts[i].Attempt.Safety.ForbiddenPaths)
		}
	}
	payload, err := json.Marshal(guestPlan)
	if err != nil {
		return base.Contract{}, err
	}
	bootstrapSnippet := toolchainBootstrapSnippetForPlan(plan)
	stepCommand := fmt.Sprintf(`%s cat <<'EOF' | base64 -d > /tmp/airlock-autofix.json
%s
EOF
chmod +x /tmp/airlock
summary_json=$(/tmp/airlock autofix-run /tmp/airlock-autofix.json)
echo "$summary_json" > /airlock/artifacts/autofix-result.json
summary_path=$(python3 -c 'import json,sys; print(json.loads(sys.stdin.read()).get("summaryPath",""))' <<<"$summary_json")
if [ -n "$summary_path" ] && [ -f "$summary_path" ]; then cp "$summary_path" /airlock/artifacts/autofix-summary.json; fi`, bootstrapSnippet, base64.StdEncoding.EncodeToString(payload))
	var c base.Contract
	c.Backend.Kind = backendKind
	c.Sandbox.NamePrefix = "autofix"
	c.Sandbox.ArtifactsDir = plan.ArtifactsDir
	c.Sandbox.CPU = 2
	c.Sandbox.MemoryGiB = 4
	c.Sandbox.DiskGiB = 20
	c.Sandbox.TTLMinutes = 60
	c.Repo.CloneURL = cloneURL
	c.Repo.Ref = headSHA
	c.Repo.Subdir = relSubdir
	c.Security.BootstrapNetwork = base.NetworkAllowlist
	c.Security.BootstrapAllowHosts = []string{"github.com", "go.dev", "dl.google.com", "proxy.golang.org", "sum.golang.org"}
	c.Security.BootstrapAptPackages = []string{"git", "curl", "ca-certificates", "python3", "build-essential"}
	c.Security.Network = base.NetworkAllowlist
	c.Security.AllowHosts = []string{"github.com", "go.dev", "dl.google.com", "proxy.golang.org", "sum.golang.org"}
	c.Security.ExportPaths = []string{"/airlock/artifacts"}
	c.Steps = []base.Step{{Name: "autofix-runner", Run: stepCommand, TimeoutSeconds: 3600}}
	return c, nil
}

func prefixPaths(prefix string, paths []string) []string {
	if prefix == "" {
		return append([]string(nil), paths...)
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.ToSlash(filepath.Join(prefix, p)))
	}
	return out
}

func toolchainBootstrapSnippet(repo string) string {
	profile, err := DetectRepo(repo)
	if err != nil {
		return ""
	}
	cmd := goToolchainBootstrapCommand(profile)
	if cmd == "" {
		return ""
	}
	return cmd + "\n"
}

// toolchainBootstrapSnippetForPlan uses the plan's explicit RepoType when set
// (e.g. when compiled from airlock fix which has already probed the repo),
// falling back to local repo detection for standalone autofix-run invocations.
func toolchainBootstrapSnippetForPlan(plan AutofixPlan) string {
	if strings.TrimSpace(plan.RepoType) == "go" {
		// Build a synthetic profile with enough info for bootstrap command
		// detection; we only need RepoType and ScopeRoot for Go toolchain.
		profile := RepoProfile{
			RepoType:  "go",
			RepoRoot:  plan.Repo,
			ScopeRoot: plan.Repo,
		}
		cmd := goToolchainBootstrapCommand(profile)
		if cmd != "" {
			return cmd + "\n"
		}
	}
	if strings.TrimSpace(plan.RepoType) != "" && strings.TrimSpace(plan.RepoType) != "go" {
		return "" // non-Go repos have no toolchain bootstrap snippet
	}
	return toolchainBootstrapSnippet(plan.Repo)
}

// resolveSymlinks resolves symlinks in path so that filepath.Rel works
// correctly across OS-level symlinks (e.g. /var → /private/var on macOS).
func resolveSymlinks(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Fall back to os.Stat-based absolute resolution
		abs, absErr := filepath.Abs(path)
		if absErr != nil {
			return path, err
		}
		return abs, nil
	}
	return resolved, nil
}

// AutofixPlanFromIssue enriches an AutofixPlan produced by synthesis with
// the issue-level context (clone URL, repo type) needed for correct VM
// contract compilation when invoked from airlock fix.
func AutofixPlanFromIssue(plan AutofixPlan, issue GitHubIssue, repoType string) AutofixPlan {
	out := plan
	if out.CloneURL == "" {
		out.CloneURL = issue.CloneURL
	}
	if out.RepoType == "" {
		out.RepoType = repoType
	}
	return out
}
