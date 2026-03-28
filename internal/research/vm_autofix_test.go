package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	base "github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/util"
)

func TestAssessRepoMarksHostToolchainBlockedVMRunnable(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-probe-toolchain-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/test\n\ngo 99.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile, err := DetectRepo(repo)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := AssessRepo(profile)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Status != "host_toolchain_blocked_vm_runnable" {
		t.Fatalf("unexpected status: %#v", assessment)
	}
	if assessment.Runnable || assessment.HostRunnable || !assessment.VMRunnable || assessment.RecommendedExecution != "vm" {
		t.Fatalf("unexpected routing assessment: %#v", assessment)
	}
}

func TestCompileAutofixPlanToVMContract(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-vm-autofix-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"go.mod": "module example.com/test\n\ngo 1.25.2\n", "flag.txt": "bad\n"}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/project.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	contract, err := CompileAutofixPlanToVMContract(AutofixPlan{
		Objective:    "test",
		Repo:         repo,
		ArtifactsDir: "/tmp/airlock-test-artifacts",
		Attempts: []AttemptFile{{
			Attempt:  AttemptSpec{Name: "flip", Validation: Phase{Command: "true", Repeat: 1}},
			Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "good\n"}},
		}},
	}, base.BackendLima)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Repo.CloneURL != "https://github.com/example/project.git" {
		t.Fatalf("unexpected clone url: %s", contract.Repo.CloneURL)
	}
	if len(contract.Steps) != 1 || !strings.Contains(contract.Steps[0].Run, "/tmp/airlock autofix-run") {
		t.Fatalf("unexpected steps: %#v", contract.Steps)
	}
	if !strings.Contains(contract.Steps[0].Run, "go1.25.2.linux-") {
		t.Fatalf("expected go bootstrap in step, got %s", contract.Steps[0].Run)
	}
}
