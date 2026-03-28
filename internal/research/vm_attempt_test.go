package research

import (
	"os"
	"strings"
	"testing"

	base "github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/util"
)

func TestCompileAttemptFileToVMContract(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-vm-attempt-")
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
	contract, err := CompileAttemptFileToVMContract(AttemptFile{
		Repo:         repo,
		ArtifactsDir: "/tmp/airlock-attempt-artifacts",
		Attempt: AttemptSpec{
			Name:       "flip",
			Validation: Phase{Command: "true", Repeat: 1},
			Safety:     SafetyBudget{AllowedPaths: []string{"flag.txt"}},
		},
		Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "good\n"}},
	}, base.BackendLima)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Repo.CloneURL != "https://github.com/example/project.git" {
		t.Fatalf("unexpected clone url: %s", contract.Repo.CloneURL)
	}
	if len(contract.Steps) != 1 || !strings.Contains(contract.Steps[0].Run, "/tmp/airlock attempt-run") {
		t.Fatalf("unexpected steps: %#v", contract.Steps)
	}
	if !strings.Contains(contract.Steps[0].Run, "go1.25.2.linux-") {
		t.Fatalf("expected go bootstrap in step, got %s", contract.Steps[0].Run)
	}
}
