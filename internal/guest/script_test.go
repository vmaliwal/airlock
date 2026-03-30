package guest

import (
	"strings"
	"testing"

	"github.com/vmaliwal/airlock/internal/contract"
)

func TestBuildScriptContainsIsolationDefaults(t *testing.T) {
	var c contract.Contract
	c.Backend.Kind = contract.BackendLima
	c.Repo.CloneURL = "https://github.com/elastic/beats.git"
	c.Repo.Ref = "main"
	c.Security.BootstrapNetwork = contract.NetworkAllowlist
	c.Security.BootstrapAllowHosts = []string{"archive.ubuntu.com", "security.ubuntu.com"}
	c.Security.BootstrapAptPackages = []string{"git", "ca-certificates"}
	c.Security.Network = contract.NetworkDeny
	c.Security.IncludePatch = true
	c.Steps = []contract.Step{{Name: "repro", Run: "go test ./...", TimeoutSeconds: 600}}

	script := BuildScript(c, "sandbox-1", map[string]string{"FOO": "bar"})
	checks := []string{
		"export HOME=\"$AIRLOCK_HOME\"",
		"export XDG_CONFIG_HOME=\"$AIRLOCK_XDG_CONFIG_HOME\"",
		"export AIRLOCK_INCLUDE_PATCH=1",
		"BOOTSTRAP_APT_PACKAGES=('git' 'ca-certificates')",
		"sudo apt-get update",
		"git clone --depth 1 --filter=blob:none 'https://github.com/elastic/beats.git' repo",
		"git -c http.https://github.com/.extraHeader=\"AUTHORIZATION: basic $auth_header\" clone --depth 1 --filter=blob:none 'https://github.com/elastic/beats.git' repo",
		"sudo iptables -P OUTPUT DROP",
		"FOO='bar'",
	}
	for _, check := range checks {
		if !strings.Contains(script, check) {
			t.Fatalf("script missing %q", check)
		}
	}
}
