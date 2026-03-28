package research

import (
	base64 "encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	base "github.com/vmaliwal/airlock/internal/contract"
)

func CompileAttemptFileToVMContract(cfg AttemptFile, backendKind base.BackendKind) (base.Contract, error) {
	gitRoot, err := GitTopLevel(cfg.Repo)
	if err != nil {
		return base.Contract{}, fmt.Errorf("detect git root: %w", err)
	}
	cloneURL, err := GitRemoteOrigin(cfg.Repo)
	if err != nil {
		return base.Contract{}, fmt.Errorf("detect origin remote: %w", err)
	}
	cloneURL = NormalizeCloneURL(cloneURL)
	headSHA, err := GitHeadSHA(gitRoot)
	if err != nil {
		return base.Contract{}, fmt.Errorf("detect head sha: %w", err)
	}
	relSubdir, err := filepath.Rel(gitRoot, cfg.Repo)
	if err != nil {
		return base.Contract{}, fmt.Errorf("resolve subdir: %w", err)
	}
	if relSubdir == "." {
		relSubdir = ""
	}
	guestCfg := cfg
	guestRepoPath := "/airlock/work/repo"
	if relSubdir != "" {
		guestRepoPath = filepath.ToSlash(filepath.Join(guestRepoPath, relSubdir))
	}
	guestCfg.Repo = guestRepoPath
	guestCfg.ArtifactsDir = "/airlock/artifacts/attempt"
	guestCfg.Checkpoint = ""
	if relSubdir != "" {
		guestCfg.Attempt.Safety.AllowedPaths = prefixPaths(relSubdir, guestCfg.Attempt.Safety.AllowedPaths)
		guestCfg.Attempt.Safety.ForbiddenPaths = prefixPaths(relSubdir, guestCfg.Attempt.Safety.ForbiddenPaths)
	}
	payload, err := json.Marshal(guestCfg)
	if err != nil {
		return base.Contract{}, err
	}
	stepCommand := fmt.Sprintf("%s cat <<'EOF' | base64 -d > /tmp/airlock-attempt.json\n%s\nEOF\nchmod +x /tmp/airlock\n/tmp/airlock attempt-run /tmp/airlock-attempt.json", toolchainBootstrapSnippet(cfg.Repo), base64.StdEncoding.EncodeToString(payload))
	var c base.Contract
	c.Backend.Kind = backendKind
	c.Sandbox.NamePrefix = "attempt"
	c.Sandbox.ArtifactsDir = cfg.ArtifactsDir
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
	c.Steps = []base.Step{{Name: "attempt-runner", Run: stepCommand, TimeoutSeconds: 3600}}
	return c, nil
}
