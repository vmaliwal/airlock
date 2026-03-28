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
	decision, err := PreflightRepo(repo, "lima")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Route != "vm" || decision.SuggestedVMBackend != "lima" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}
