package firecracker

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vmaliwal/airlock/internal/backend"
	"github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/env"
	"github.com/vmaliwal/airlock/internal/guest"
	"github.com/vmaliwal/airlock/internal/util"
)

type Backend struct{}

func (b Backend) Kind() contract.BackendKind { return contract.BackendFirecracker }

func (b Backend) CheckPrereqs() []string {
	errs := []string{}
	if runtime.GOOS != "linux" {
		errs = append(errs, "local firecracker backend requires a Linux host")
	}
	if !util.CommandExists("ssh") {
		errs = append(errs, "ssh not found on PATH")
	}
	if !util.CommandExists("scp") {
		errs = append(errs, "scp not found on PATH")
	}
	if !util.CommandExists("airlock-firecracker-host.sh") {
		errs = append(errs, "airlock-firecracker-host.sh not found on PATH")
	}
	return errs
}

func (b Backend) Run(c contract.Contract) (backend.RunResult, error) {
	if needsGuestBinaryInjection(c) {
		return backend.RunResult{}, fmt.Errorf("firecracker backend does not yet support guest binary injection for /tmp/airlock or /tmp/airlock-researchguest steps")
	}
	fc := c.Backend.FirecrackerHost
	if fc == nil {
		return backend.RunResult{}, fmt.Errorf("missing firecracker host config")
	}
	sandboxName := util.SafeName(fmt.Sprintf("%s-%d", c.Sandbox.NamePrefix, time.Now().Unix()))
	if err := util.EnsureDir(c.Sandbox.ArtifactsDir); err != nil {
		return backend.RunResult{}, err
	}
	workDir, err := os.MkdirTemp("", "airlock-firecracker-")
	if err != nil {
		return backend.RunResult{}, err
	}
	defer os.RemoveAll(workDir)

	allowedEnv := env.BuildGuestEnv(util.EnvMapFromSlice(os.Environ()), c.Security.AllowedEnv)
	script := guest.BuildScript(c, sandboxName, allowedEnv)
	scriptPath := filepath.Join(workDir, "guest-run.sh")
	if err := util.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return backend.RunResult{}, err
	}

	if fc.Mode == "local" {
		if !util.CommandExists("airlock-firecracker-host.sh") {
			return backend.RunResult{}, fmt.Errorf("airlock-firecracker-host.sh not found on PATH")
		}
		if _, err := util.RunLocal("airlock-firecracker-host.sh", []string{"run", "--name", sandboxName, "--contract", scriptPath, "--artifacts", c.Sandbox.ArtifactsDir}, util.RunOptions{}); err != nil {
			return backend.RunResult{}, err
		}
		return backend.RunResult{SummaryPath: filepath.Join(c.Sandbox.ArtifactsDir, sandboxName+"-summary.json")}, nil
	}

	remoteDir := filepath.Join(fc.RemoteWorkDir, sandboxName)
	sshTarget := fmt.Sprintf("%s@%s", fc.User, fc.Host)
	sshArgs := sshBaseArgs(fc)
	if _, err := util.RunLocal("ssh", append(sshArgs, sshTarget, "mkdir", "-p", remoteDir), util.RunOptions{}); err != nil {
		return backend.RunResult{}, fmt.Errorf("prepare remote dir: %w", err)
	}
	if _, err := util.RunLocal("scp", append(scpBaseArgs(fc), scriptPath, sshTarget+":"+remoteDir+"/guest-run.sh"), util.RunOptions{}); err != nil {
		return backend.RunResult{}, fmt.Errorf("upload guest script: %w", err)
	}
	remoteCmd := fmt.Sprintf("cd %s && airlock-firecracker-host.sh run --name %s --contract %s/guest-run.sh --artifacts %s", shell(remoteDir), shell(sandboxName), shell(remoteDir), shell(remoteDir))
	if _, err := util.RunLocal("ssh", append(sshArgs, sshTarget, remoteCmd), util.RunOptions{}); err != nil {
		return backend.RunResult{}, fmt.Errorf("run remote firecracker host shim: %w", err)
	}
	hostSummary := filepath.Join(c.Sandbox.ArtifactsDir, sandboxName+"-summary.json")
	if _, err := util.RunLocal("scp", append(scpBaseArgs(fc), sshTarget+":"+remoteDir+"/summary.json", hostSummary), util.RunOptions{}); err != nil {
		return backend.RunResult{}, fmt.Errorf("copy summary back: %w", err)
	}
	return backend.RunResult{SummaryPath: hostSummary}, nil
}

func sshBaseArgs(fc *contract.FirecrackerHostConfig) []string {
	args := []string{}
	if fc.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", fc.Port))
	}
	if fc.SSHIdentityFile != "" {
		args = append(args, "-i", fc.SSHIdentityFile)
	}
	return args
}

func scpBaseArgs(fc *contract.FirecrackerHostConfig) []string {
	args := []string{}
	if fc.Port > 0 {
		args = append(args, "-P", fmt.Sprintf("%d", fc.Port))
	}
	if fc.SSHIdentityFile != "" {
		args = append(args, "-i", fc.SSHIdentityFile)
	}
	return args
}

func shell(s string) string {
	return "'" + s + "'"
}

func needsGuestBinaryInjection(c contract.Contract) bool {
	for _, step := range c.Steps {
		if strings.Contains(step.Run, "/tmp/airlock") || strings.Contains(step.Run, "/tmp/airlock-researchguest") {
			return true
		}
	}
	return false
}
