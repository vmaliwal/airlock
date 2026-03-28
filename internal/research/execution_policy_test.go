package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecideExecutionPolicyRoutesNodeRepoToVMByDefault(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-policy-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte("{\"name\":\"example\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := DecideExecutionPolicy(repo, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Preflight.Route != "vm" || out.BackendKind != "lima" {
		t.Fatalf("unexpected policy decision: %#v", out)
	}
}

func TestDecideExecutionPolicyAllowsExplicitHostException(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-policy-host-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte("{\"name\":\"example\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "package-lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := DecideExecutionPolicy(repo, "lima", true)
	if err != nil {
		t.Fatal(err)
	}
	if out.Preflight.Route != "host" || !out.HostExecutionExceptionUsed {
		t.Fatalf("unexpected policy decision: %#v", out)
	}
}
