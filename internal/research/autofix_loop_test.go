package research

import (
	"path/filepath"
	"testing"
)

func TestRunAutofixLoopStopsOnNoNewAttempts(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{"flag.txt": "bad\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts := filepath.Join(repo, "autofix-loop-artifacts")
	plan := AutofixPlan{
		Objective:    "flip bad to good",
		Repo:         repo,
		ArtifactsDir: artifacts,
		Attempts: []AttemptFile{{
			Attempt: AttemptSpec{
				Name:       "wrong-fix",
				Validation: Phase{Command: `grep -q '^good$' flag.txt`, Repeat: 1, Success: SuccessRule{ExitCode: pint(0), MinPassRate: pfloat(1)}},
				Safety:     SafetyBudget{MaxFilesChanged: 1, AllowedPaths: []string{"flag.txt"}},
			},
			Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "stillbad\n"}},
		}},
	}
	loop, err := RunAutofixLoop(AutofixLoopPolicy{MaxRounds: 2}, func(round int, previous *AutofixSummary) (*AutofixPlan, error) {
		return &plan, nil
	}, nil)
	if err == nil {
		t.Fatal("expected no-new-attempts loop failure")
	}
	if len(loop.Rounds) != 2 || loop.Rounds[1].StopReason != "no_new_attempts" {
		t.Fatalf("unexpected loop summary: %#v", loop)
	}
}

func TestRunAutofixLoopFindsWinnerInSecondRound(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{"flag.txt": "bad\n"}); err != nil {
		t.Fatal(err)
	}
	artifacts := filepath.Join(repo, "autofix-loop-win-artifacts")
	wrong := AttemptFile{
		Attempt: AttemptSpec{
			Name:       "wrong-fix",
			Validation: Phase{Command: `grep -q '^good$' flag.txt`, Repeat: 1, Success: SuccessRule{ExitCode: pint(0), MinPassRate: pfloat(1)}},
			Safety:     SafetyBudget{MaxFilesChanged: 1, AllowedPaths: []string{"flag.txt"}},
		},
		Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "stillbad\n"}},
	}
	good := AttemptFile{
		Attempt: AttemptSpec{
			Name:       "good-fix",
			Validation: Phase{Command: `grep -q '^good$' flag.txt`, Repeat: 1, Success: SuccessRule{ExitCode: pint(0), MinPassRate: pfloat(1)}},
			Safety:     SafetyBudget{MaxFilesChanged: 1, AllowedPaths: []string{"flag.txt"}},
		},
		Mutation: MutationSpec{SearchReplace: &SearchReplaceMutation{Path: "flag.txt", OldText: "bad\n", NewText: "good\n"}},
	}
	loop, err := RunAutofixLoop(AutofixLoopPolicy{MaxRounds: 2}, func(round int, previous *AutofixSummary) (*AutofixPlan, error) {
		plan := AutofixPlan{Objective: "flip bad to good", Repo: repo, ArtifactsDir: artifacts}
		if round == 1 {
			plan.Attempts = []AttemptFile{wrong}
		} else {
			plan.Attempts = []AttemptFile{wrong, good}
		}
		return &plan, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !loop.Success || loop.WinningAttempt != "good-fix" {
		t.Fatalf("unexpected loop summary: %#v", loop)
	}
	if len(loop.Rounds) != 2 || !loop.Rounds[1].Success {
		t.Fatalf("expected second-round success: %#v", loop)
	}
	if loop.Rounds[1].PromotedCheckpoint == "" {
		t.Fatalf("expected promoted checkpoint on winning round: %#v", loop)
	}
}
