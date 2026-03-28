package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunAttemptFileSearchReplace(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-attemptfile-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"flag.txt": "bad\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts := filepath.Join(repo, "artifacts")
	zero := 0
	one := 1.0
	outcome, err := RunAttemptFile(AttemptFile{
		Repo:         repo,
		ArtifactsDir: artifacts,
		Attempt: AttemptSpec{
			Name: "search-replace",
			Validation: Phase{
				Command: `grep -q '^good$' flag.txt`,
				Repeat:  1,
				Success: SuccessRule{ExitCode: &zero, MinPassRate: &one},
			},
			Safety: SafetyBudget{MaxFilesChanged: 1, AllowedPaths: []string{"flag.txt"}},
		},
		Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "good\n"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Success {
		t.Fatalf("expected success, got %#v", outcome)
	}
	if _, err := os.Stat(filepath.Join(artifacts, "lessons.jsonl")); err != nil {
		t.Fatal(err)
	}
}
