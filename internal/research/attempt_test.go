package research

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunNativeAttemptSuccess(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-attempt-success-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"flag.txt": "bad\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := os.MkdirTemp("", "airlock-attempt-artifacts-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(artifacts)
	base, err := GitHeadSHA(repo)
	if err != nil {
		t.Fatal(err)
	}
	zero := 0
	one := 1.0
	outcome, err := RunNativeAttempt(repo, artifacts, base, AttemptSpec{
		Name: "flip-flag",
		MutationCommand: `python3 - <<'PY'
from pathlib import Path
p = Path('flag.txt')
p.write_text('good\n')
PY`,
		Validation: Phase{
			Command: `grep -q '^good$' flag.txt`,
			Repeat:  1,
			Success: SuccessRule{ExitCode: &zero, MinPassRate: &one},
		},
		Safety:        SafetyBudget{MaxFilesChanged: 1, MaxLocChanged: 4, AllowedPaths: []string{"flag.txt"}},
		CommitMessage: "attempt: flip flag",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Success {
		t.Fatalf("expected success, got %#v", outcome)
	}
	if outcome.DiffStats.FilesChangedCount != 1 {
		t.Fatalf("expected one changed file, got %#v", outcome.DiffStats)
	}
	if outcome.PatchPath == "" {
		t.Fatal("expected patch path")
	}
	dirty, err := GitIsDirty(repo)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("expected clean repo after committed success")
	}
}

func TestRunNativeAttemptResetsFailedValidation(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-attempt-fail-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"flag.txt": "bad\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts, err := os.MkdirTemp("", "airlock-attempt-artifacts-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(artifacts)
	base, err := GitHeadSHA(repo)
	if err != nil {
		t.Fatal(err)
	}
	zero := 0
	one := 1.0
	outcome, err := RunNativeAttempt(repo, artifacts, base, AttemptSpec{
		Name: "still-bad",
		MutationCommand: `python3 - <<'PY'
from pathlib import Path
p = Path('flag.txt')
p.write_text('stillbad\n')
PY`,
		Validation: Phase{
			Command: `grep -q '^good$' flag.txt`,
			Repeat:  1,
			Success: SuccessRule{ExitCode: &zero, MinPassRate: &one},
		},
		Safety:  SafetyBudget{MaxFilesChanged: 1, AllowedPaths: []string{"flag.txt"}},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Success {
		t.Fatalf("expected validation failure, got %#v", outcome)
	}
	dirty, err := GitIsDirty(repo)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("expected repo reset after failed validation")
	}
	data, err := os.ReadFile(filepath.Join(repo, "flag.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "bad\n" {
		t.Fatalf("expected file reset to checkpoint, got %q", string(data))
	}
}
