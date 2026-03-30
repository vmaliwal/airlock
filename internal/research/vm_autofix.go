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
	cloneURL, err := GitRemoteOrigin(plan.Repo)
	if err != nil {
		return base.Contract{}, fmt.Errorf("detect origin remote: %w", err)
	}
	cloneURL = NormalizeCloneURL(cloneURL)
	headSHA, err := GitHeadSHA(gitRoot)
	if err != nil {
		return base.Contract{}, fmt.Errorf("detect head sha: %w", err)
	}
	relSubdir, err := filepath.Rel(gitRoot, plan.Repo)
	if err != nil {
		return base.Contract{}, fmt.Errorf("resolve subdir: %w", err)
	}
	if relSubdir == "." {
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
	stepCommand := fmt.Sprintf(`%s cat <<'EOF' | base64 -d > /tmp/airlock-autofix.json
%s
EOF
chmod +x /tmp/airlock
summary_json=$(/tmp/airlock autofix-run /tmp/airlock-autofix.json)
echo "$summary_json" > /airlock/artifacts/autofix-result.json
summary_path=$(python3 -c 'import json,sys; print(json.loads(sys.stdin.read()).get("summaryPath",""))' <<<"$summary_json")
if [ -n "$summary_path" ] && [ -f "$summary_path" ]; then cp "$summary_path" /airlock/artifacts/autofix-summary.json; fi`, toolchainBootstrapSnippet(plan.Repo), base64.StdEncoding.EncodeToString(payload))
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
