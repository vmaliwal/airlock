package research

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vmaliwal/airlock/internal/util"
)

func TestRunPlannerEvalCases(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"go.mod":     "module example.com/beats\n\ngo 1.23\n",
		"version.go": "package beats\n\nfunc alias() string {\n\treturn \"V4_0_0_0\"\n}\n",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/beats.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	expected := true
	summary, err := RunPlannerEvalCases([]PlannerEvalCase{{
		Name:                  "beats-alias",
		Input:                 PlanInput{RepoPath: repo, FailureText: "expected V4_1_0_0, got V4_0_0_0", FailingCommand: "go test ./... -run TestKafkaAlias -count=1"},
		ExpectedSupported:     &expected,
		ExpectedMutationKinds: []string{"replace_line"},
	}}, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CaseCount != 1 || summary.Top1HitCount != 1 || summary.SchemaValidCount != 1 {
		data, _ := json.MarshalIndent(summary, "", "  ")
		t.Fatalf("unexpected eval summary: %s", string(data))
	}
	if len(summary.Results) != 1 || len(summary.Results[0].Errors) != 0 {
		data, _ := json.MarshalIndent(summary, "", "  ")
		t.Fatalf("unexpected eval case result: %s", string(data))
	}
}

func TestLoadPlannerEvalCases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cases.json")
	if err := os.WriteFile(path, []byte(`{"cases":[{"name":"x","input":{"repoPath":"/tmp/repo"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cases, err := LoadPlannerEvalCases(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Name != "x" {
		t.Fatalf("unexpected cases: %#v", cases)
	}
}
