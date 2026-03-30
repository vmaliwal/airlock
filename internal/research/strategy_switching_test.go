package research

import (
	"path/filepath"
	"testing"

	"github.com/vmaliwal/airlock/internal/util"
)

func TestPlannerRequestForAvoidsFailedMutationKindsWhenAlternativesExist(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{"go.mod": "module example.com/x\n\ngo 1.25.8\n", "x.go": "package x\n"}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/x.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	input := PlanInput{RepoPath: repo, Notes: "## Prior round memory\nfailed_mutation_kinds: replace_line, nil_guard"}
	report, err := PlanFromInput(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	req := plannerRequestFor(input, report, "go test ./...")
	for _, kind := range req.AllowedMutations {
		if kind == "replace_line" || kind == "nil_guard" {
			t.Fatalf("expected failed mutation kinds to be filtered, got %#v", req.AllowedMutations)
		}
	}
}

func TestSynthesizeAutofixPlanAvoidsFailedMutationKinds(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"go.mod":          "module example.com/beats\n\ngo 1.25.8\n",
		"version.go":      "package beats\n\nfunc alias() string {\n\treturn \"V4_0_0_0\"\n}\n",
		"version_test.go": "package beats\n",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/beats.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	input := PlanInput{
		RepoPath:       repo,
		FailureText:    "expected V4_1_0_0, got V4_0_0_0",
		Notes:          "## Prior round memory\nfailed_mutation_kinds: replace_line",
		FailingCommand: "go test ./...",
	}
	report, err := SynthesizeAutofixPlan(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Attempts) == 0 {
		t.Fatal("expected synthesized attempts")
	}
	for _, attempt := range report.Attempts {
		if attempt.MutationKind == "replace_line" {
			t.Fatalf("expected round steering to avoid replace_line, got %#v", report.Attempts)
		}
	}
	if report.Attempts[0].MutationKind != "search_replace" {
		t.Fatalf("expected fallback mutation family, got %#v", report.Attempts)
	}
	if report.Attempts[0].Attempt.Mutation.SearchReplace == nil {
		t.Fatalf("expected search_replace mutation, got %#v", report.Attempts[0].Attempt.Mutation)
	}
	if report.Attempts[0].Attempt.Mutation.SearchReplace.Path != filepath.ToSlash("version.go") {
		t.Fatalf("unexpected mutation path: %#v", report.Attempts[0].Attempt.Mutation.SearchReplace)
	}
}
