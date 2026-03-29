package research

import (
	"os"
	"path/filepath"
	"testing"

	base "github.com/vmaliwal/airlock/internal/contract"
)

func TestExecuteRunContractSetupCommitNoopDoesNotFail(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-exec-setup-noop-repo-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"a.txt": "ok\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := os.MkdirTemp("", "airlock-exec-setup-noop-artifacts-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(artifacts)
	var rc RunContract
	rc.Objective = "test"
	rc.Mode = "read_only"
	rc.Airlock.Backend.Kind = base.BackendLima
	rc.Setup = []SetupStep{{Name: "noop setup", Command: "true", CommitMessage: "checkpoint setup"}}
	rc.Reproduction = Phase{Command: "false", Repeat: 1, Success: SuccessRule{MinFailures: pintVal(1)}}
	rc.Validation = ValidationSpec{TargetCommand: "true", Repeat: 1, Success: SuccessRule{ExitCode: pintVal(0)}}
	if err := ExecuteRunContract(rc, repo, artifacts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(artifacts, "attempt-log.jsonl")); err != nil {
		t.Fatalf("expected attempt log artifact, got %v", err)
	}
}

func TestExecuteRunContractWritesExecutionPolicy(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-exec-policy-repo-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"a.txt": "ok\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := os.MkdirTemp("", "airlock-exec-policy-artifacts-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(artifacts)
	var rc RunContract
	rc.Objective = "test"
	rc.Mode = "read_only"
	rc.HostExecutionException = true
	rc.Airlock.Backend.Kind = base.BackendLima
	rc.Reproduction = Phase{Command: "false", Repeat: 1, Success: SuccessRule{MinFailures: pintVal(1)}}
	rc.Validation = ValidationSpec{TargetCommand: "true", Repeat: 1, Success: SuccessRule{ExitCode: pintVal(0)}}
	if err := ExecuteRunContract(rc, repo, artifacts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(artifacts, "execution-policy.json")); err != nil {
		t.Fatalf("expected execution-policy artifact, got %v", err)
	}
}
