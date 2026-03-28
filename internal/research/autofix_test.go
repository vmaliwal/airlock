package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunAutofixPlanStopsOnWinningAttempt(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-autofix-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := InitTempGitRepo(repo, map[string]string{"flag.txt": "bad\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts := filepath.Join(repo, "autofix-artifacts")
	plan := AutofixPlan{
		Objective:    "flip bad to good",
		Repo:         repo,
		ArtifactsDir: artifacts,
		Attempts: []AttemptFile{
			{
				Attempt: AttemptSpec{
					Name:       "wrong-fix",
					Validation: Phase{Command: `grep -q '^good$' flag.txt`, Repeat: 1, Success: SuccessRule{ExitCode: pint(0), MinPassRate: pfloat(1)}},
					Safety:     SafetyBudget{MaxFilesChanged: 1, AllowedPaths: []string{"flag.txt"}},
				},
				Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "stillbad\n"}},
			},
			{
				Attempt: AttemptSpec{
					Name:          "good-fix",
					Validation:    Phase{Command: `grep -q '^good$' flag.txt`, Repeat: 1, Success: SuccessRule{ExitCode: pint(0), MinPassRate: pfloat(1)}},
					Safety:        SafetyBudget{MaxFilesChanged: 1, AllowedPaths: []string{"flag.txt"}},
					CommitMessage: "attempt: good fix",
				},
				Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "good\n"}},
			},
		},
	}
	summaryPath, err := RunAutofixPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	var summary AutofixSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if !summary.Success || summary.WinningAttempt != "good-fix" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}
