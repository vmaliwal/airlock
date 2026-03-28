package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanRepoUsesLessonRootAndRanksKinds(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-plan-repo-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/test\n\ngo 1.22.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lessonsRoot, err := os.MkdirTemp("", "airlock-plan-lessons-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(lessonsRoot)
	if err := os.WriteFile(filepath.Join(lessonsRoot, "lessons.jsonl"), []byte(`{"repo":"`+repo+`","attemptName":"a","mutationKind":"apply_patch","success":true}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv(LessonsRootEnv)
	defer os.Setenv(LessonsRootEnv, old)
	if err := os.Setenv(LessonsRootEnv, lessonsRoot); err != nil {
		t.Fatal(err)
	}
	report, err := PlanRepo(repo, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Investigation.Preflight.Route != "vm" {
		t.Fatalf("expected vm route, got %#v", report.Investigation.Preflight)
	}
	if len(report.RankedMutationKinds) == 0 || report.RankedMutationKinds[0].Kind != "apply_patch" {
		t.Fatalf("expected apply_patch to rank first, got %#v", report.RankedMutationKinds)
	}
}

func TestPlanFromInputCarriesBugSignal(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-plan-input-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte("{\"name\":\"example\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := PlanFromInput(PlanInput{
		RepoPath:       repo,
		IssueURL:       "https://github.com/example/repo/issues/1",
		FailingCommand: "npm test -- broken",
		FailureText:    "timeout exceeded",
	}, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Input.IssueURL == "" || len(report.CandidateCommands) == 0 {
		t.Fatalf("expected bug signal to carry into report: %#v", report)
	}
}
