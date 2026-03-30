package research

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func goToolchainBootstrapCommand(profile RepoProfile) string {
	if profile.RepoType != "go" {
		return ""
	}
	goModPath := filepath.Join(profile.ScopeRoot, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		goModPath = filepath.Join(profile.RepoRoot, "go.mod")
	}
	goModBytes, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	req := detectGoDirective(string(goModBytes))
	if req == "" {
		return ""
	}
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	return fmt.Sprintf("mkdir -p /tmp/airlock-go && if ! command -v go >/dev/null 2>&1 || ! go version | grep -q 'go%s'; then curl -fsSL https://go.dev/dl/go%s.linux-%s.tar.gz -o /tmp/go.tgz && rm -rf /tmp/airlock-go/go && tar -C /tmp/airlock-go -xzf /tmp/go.tgz; fi && export PATH=/tmp/airlock-go/go/bin:$PATH && export GOTOOLCHAIN=local", req, req, arch)
}

func applyBootstrapPrefix(cmd, prefix string) string {
	cmd = strings.TrimSpace(cmd)
	prefix = strings.TrimSpace(prefix)
	if cmd == "" || prefix == "" {
		return cmd
	}
	return prefix + " && " + cmd
}
