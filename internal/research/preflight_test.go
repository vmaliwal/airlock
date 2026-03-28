package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreflightRepoRoutesToVMForToolchainBlockedRepo(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-preflight-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/test\n\ngo 99.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	decision, err := PreflightRepo(repo, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Route != "vm" || decision.SuggestedVMBackend != "lima" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestPreflightRepoBlocksHostExecutionByPolicy(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-preflight-host-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte("{\"name\":\"example\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	decision, err := PreflightRepo(repo, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Route != "vm" {
		t.Fatalf("expected vm route, got %#v", decision)
	}
}

func TestPreflightRepoAllowsExplicitHostException(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-preflight-explicit-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte("{\"name\":\"example\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	decision, err := PreflightRepo(repo, "lima", true)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Route != "host" {
		t.Fatalf("expected host route, got %#v", decision)
	}
}
