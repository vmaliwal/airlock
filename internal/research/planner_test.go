package research

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vmaliwal/airlock/internal/util"
)

type stubPlanner struct {
	resp PlannerResponse
	err  error
}

func (s stubPlanner) Synthesize(context.Context, PlannerRequest) (PlannerResponse, error) {
	return s.resp, s.err
}

func TestSynthesizeWithPlannerNormalizesStructuredAttempts(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"pkg/thing/go.mod":          "module example.com/thing\n\ngo 1.22\n",
		"pkg/thing/version.go":      "package thing\n\nfunc versionAlias(v string) string {\n\tif v == \"V4_0_0_0\" {\n\t\treturn \"V4_0_0_0\"\n\t}\n\treturn v\n}\n",
		"pkg/thing/version_test.go": "package thing\n",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/thing.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	input := PlanInput{
		RepoPath:       filepath.Join(repo, "pkg", "thing"),
		FailingCommand: "go test ./... -run TestKafkaAlias -count=1",
		FailureText:    "expected V4_1_0_0, got V4_0_0_0",
		Notes:          "test failure: TestKafkaAlias",
	}
	report, err := PlanFromInput(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	attempts, summary, err := synthesizeWithPlanner(context.Background(), stubPlanner{resp: PlannerResponse{
		Summary: "generated bounded alias mapping candidates",
		Attempts: []PlannerAttemptProposal{{
			Name:         "update-kafka-alias",
			MutationKind: "replace_line",
			Confidence:   "medium",
			Rationale:    "the returned alias is outdated",
			Path:         "version.go",
			OldLine:      "\t\treturn \"V4_0_0_0\"",
			NewLine:      "\t\treturn \"V4_1_0_0\"",
		}},
	}}, input, report, "go test ./... -run TestKafkaAlias -count=1")
	if err != nil {
		t.Fatal(err)
	}
	if summary == "" || len(attempts) != 1 {
		t.Fatalf("expected one normalized planner attempt, got summary=%q attempts=%d", summary, len(attempts))
	}
	m := attempts[0].Attempt.Mutation.ReplaceLine
	if m == nil || m.Path != "version.go" || m.NewLine != "\t\treturn \"V4_1_0_0\"" {
		t.Fatalf("unexpected normalized mutation: %#v", attempts[0])
	}
}

func TestNormalizePlannerAttemptsRejectsPathEscape(t *testing.T) {
	profile := RepoProfile{TargetPath: t.TempDir()}
	_, err := normalizePlannerAttempts(PlannerResponse{Attempts: []PlannerAttemptProposal{{
		Name:         "bad",
		MutationKind: "ensure_line",
		Confidence:   "low",
		Rationale:    "bad path",
		Path:         "../escape.go",
		Line:         "boom",
	}}}, profile, "go test ./...")
	if err == nil {
		t.Fatal("expected path validation error")
	}
}
