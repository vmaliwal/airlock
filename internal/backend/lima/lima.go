package lima

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vmaliwal/airlock/internal/backend"
	"github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/env"
	"github.com/vmaliwal/airlock/internal/guest"
	"github.com/vmaliwal/airlock/internal/util"
)

type Backend struct{}

func (b Backend) Kind() contract.BackendKind { return contract.BackendLima }

func (b Backend) CheckPrereqs() []string {
	var errs []string
	if !util.CommandExists("limactl") {
		errs = append(errs, "limactl not found on PATH")
	}
	return errs
}

func (b Backend) Run(c contract.Contract) (backend.RunResult, error) {
	sandboxName := util.SafeName(fmt.Sprintf("%s-%d", c.Sandbox.NamePrefix, time.Now().Unix()))
	if err := util.EnsureDir(c.Sandbox.ArtifactsDir); err != nil {
		return backend.RunResult{}, err
	}
	workDir, err := os.MkdirTemp("", "airlock-lima-")
	if err != nil {
		return backend.RunResult{}, err
	}
	defer os.RemoveAll(workDir)

	cfgPath := filepath.Join(workDir, "lima.yaml")
	if err := util.WriteFile(cfgPath, []byte(buildConfig(c, sandboxName)), 0o644); err != nil {
		return backend.RunResult{}, err
	}

	allowedEnv := env.BuildGuestEnv(util.EnvMapFromSlice(os.Environ()), c.Security.AllowedEnv)
	script := guest.BuildScript(c, sandboxName, allowedEnv)
	scriptPath := filepath.Join(workDir, "guest-run.sh")
	if err := util.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return backend.RunResult{}, err
	}

	if _, err := util.RunLocal("limactl", []string{"start", "--name", sandboxName, cfgPath}, util.RunOptions{}); err != nil {
		return backend.RunResult{}, fmt.Errorf("start lima instance: %w", err)
	}
	defer util.RunLocal("limactl", []string{"delete", "-f", sandboxName}, util.RunOptions{})

	if _, err := util.RunLocal("limactl", []string{"copy", scriptPath, sandboxName + ":/tmp/guest-run.sh"}, util.RunOptions{}); err != nil {
		return backend.RunResult{}, fmt.Errorf("copy guest script: %w", err)
	}
	if needsResearchGuestBinary(c) {
		guestBin := filepath.Join(workDir, "airlock-researchguest")
		if _, err := util.RunLocal("go", []string{"build", "-o", guestBin, "./cmd/researchguest"}, util.RunOptions{Cwd: repoRoot(), Env: guestBuildEnv("arm64")}); err != nil {
			return backend.RunResult{}, fmt.Errorf("build research guest binary: %w", err)
		}
		if _, err := util.RunLocal("limactl", []string{"copy", guestBin, sandboxName + ":/tmp/airlock-researchguest"}, util.RunOptions{}); err != nil {
			return backend.RunResult{}, fmt.Errorf("copy research guest binary: %w", err)
		}
	}
	if needsAirlockBinary(c) {
		guestBin := filepath.Join(workDir, "airlock")
		if _, err := util.RunLocal("go", []string{"build", "-o", guestBin, "./cmd/airlock"}, util.RunOptions{Cwd: repoRoot(), Env: guestBuildEnv("arm64")}); err != nil {
			return backend.RunResult{}, fmt.Errorf("build airlock guest binary: %w", err)
		}
		if _, err := util.RunLocal("limactl", []string{"copy", guestBin, sandboxName + ":/tmp/airlock"}, util.RunOptions{}); err != nil {
			return backend.RunResult{}, fmt.Errorf("copy airlock guest binary: %w", err)
		}
	}
	shellOut, shellErr := util.RunLocal("limactl", []string{"shell", sandboxName, "bash", "-lc", "chmod +x /tmp/guest-run.sh /tmp/airlock-researchguest /tmp/airlock 2>/dev/null || chmod +x /tmp/guest-run.sh && sudo mkdir -p /airlock/artifacts /airlock/work /airlock/home /airlock/xdg/config /airlock/xdg/cache /airlock/xdg/data /airlock/tmp && sudo chown -R $(id -u):$(id -g) /airlock && /tmp/guest-run.sh"}, util.RunOptions{})
	hostShellLog := filepath.Join(c.Sandbox.ArtifactsDir, sandboxName+"-guest-shell.log")
	_ = os.WriteFile(hostShellLog, shellOut, 0o644)

	hostSummary := filepath.Join(c.Sandbox.ArtifactsDir, sandboxName+"-summary.json")
	if _, err := util.RunLocal("limactl", []string{"copy", sandboxName + ":/airlock/artifacts/summary.json", hostSummary}, util.RunOptions{}); err != nil {
		if shellErr != nil {
			return backend.RunResult{}, fmt.Errorf("guest run failed before summary creation: %w", shellErr)
		}
		return backend.RunResult{}, fmt.Errorf("copy summary back: %w", err)
	}

	// export the full artifact directory as a tarball for richer research runs
	_, _ = util.RunLocal("limactl", []string{"shell", sandboxName, "bash", "-lc", "tar -C /airlock -czf /tmp/airlock-artifacts.tgz artifacts"}, util.RunOptions{})
	hostTarball := filepath.Join(c.Sandbox.ArtifactsDir, sandboxName+"-artifacts.tgz")
	_, _ = util.RunLocal("limactl", []string{"copy", sandboxName + ":/tmp/airlock-artifacts.tgz", hostTarball}, util.RunOptions{})

	// copy common artifacts if present
	for _, name := range []string{"repo.patch", "steps.json", "outcome.md", "reproduction-results.json", "validation-results.json", "baseline-results.json", "campaign-baseline.json", "attempt-log.jsonl", "autofix-result.json", "autofix-summary.json", "proof-state.json", "advancement-decision.json"} {
		hostPath := filepath.Join(c.Sandbox.ArtifactsDir, sandboxName+"-"+name)
		_, _ = util.RunLocal("limactl", []string{"copy", sandboxName + ":/airlock/artifacts/" + name, hostPath}, util.RunOptions{})
	}
	for _, step := range c.Steps {
		for _, suffix := range []string{"stdout.log", "stderr.log"} {
			name := fmt.Sprintf("%s.%s", step.Name, suffix)
			hostPath := filepath.Join(c.Sandbox.ArtifactsDir, sandboxName+"-"+name)
			_, _ = util.RunLocal("limactl", []string{"copy", sandboxName + ":/airlock/artifacts/" + name, hostPath}, util.RunOptions{})
		}
	}

	return backend.RunResult{SummaryPath: hostSummary}, nil
}

func needsResearchGuestBinary(c contract.Contract) bool {
	for _, step := range c.Steps {
		if strings.Contains(step.Run, "/tmp/airlock-researchguest") {
			return true
		}
	}
	return false
}

func needsAirlockBinary(c contract.Contract) bool {
	for _, step := range c.Steps {
		if strings.Contains(step.Run, "/tmp/airlock") {
			return true
		}
	}
	return false
}

func guestBuildEnv(goarch string) []string {
	toolchain := os.Getenv("GOTOOLCHAIN")
	if toolchain == "" {
		toolchain = "auto"
	}
	return append(os.Environ(), "GOOS=linux", "GOARCH="+goarch, "CGO_ENABLED=0", "GOTOOLCHAIN="+toolchain)
}

func repoRoot() string {
	wd, _ := os.Getwd()
	return wd
}

func buildConfig(c contract.Contract, sandboxName string) string {
	var b strings.Builder
	b.WriteString("vmType: vz\n")
	b.WriteString("images:\n")
	b.WriteString("  - location: \"https://cloud-images.ubuntu.com/minimal/releases/noble/release/ubuntu-24.04-minimal-cloudimg-arm64.img\"\n")
	b.WriteString("    arch: \"aarch64\"\n")
	b.WriteString("cpus: ")
	b.WriteString(fmt.Sprintf("%d\n", c.Sandbox.CPU))
	b.WriteString("memory: \"")
	b.WriteString(fmt.Sprintf("%dGiB\"\n", c.Sandbox.MemoryGiB))
	b.WriteString("disk: \"")
	b.WriteString(fmt.Sprintf("%dGiB\"\n", c.Sandbox.DiskGiB))
	b.WriteString("mounts: []\n")
	b.WriteString("containerd:\n")
	b.WriteString("  system: false\n")
	b.WriteString("  user: false\n")
	b.WriteString("ssh:\n")
	b.WriteString("  loadDotSSHPubKeys: false\n")
	b.WriteString("provision: []\n")
	_ = sandboxName
	return b.String()
}
