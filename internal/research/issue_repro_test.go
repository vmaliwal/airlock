package research

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vmaliwal/airlock/internal/util"
)

func TestExtractIssueReproFileGoSnippet(t *testing.T) {
	notes := "## Reproduction Steps\n1. From repo root, create a minimal test:\n\n```go\n// libbeat/common/kafka/zzz_repro_test.go\npackage kafka\n\nimport \"testing\"\n\nfunc TestRepro(t *testing.T) {\n\tt.Fatal(\"boom\")\n}\n```\n"
	path, contents, ok := extractIssueReproFile(notes)
	if !ok {
		t.Fatal("expected repro file to be extracted")
	}
	if path != "libbeat/common/kafka/zzz_repro_test.go" {
		t.Fatalf("unexpected path: %s", path)
	}
	if !strings.Contains(contents, "func TestRepro") {
		t.Fatalf("unexpected contents: %s", contents)
	}
}

func TestCompilePlanInputToRunContractIncludesIssueReproSetup(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"go.mod":     "module example.com/beats\n\ngo 1.25.8\n",
		"version.go": "package kafka\n",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/beats.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	input := PlanInput{
		RepoPath:       repo,
		IssueURL:       "https://github.com/elastic/beats/issues/49491",
		FailingCommand: "go test ./libbeat/common/kafka -run TestRepro_MajorVersionAliasUsesLatestMinor -count=1",
		FailureText:    "Kafka major alias resolves to wrong version",
		Notes:          "```go\n// libbeat/common/kafka/zzz_repro_test.go\npackage kafka\n\nimport \"testing\"\n\nfunc TestRepro_MajorVersionAliasUsesLatestMinor(t *testing.T) {\n\tt.Fatal(\"expected alias 4 to resolve to latest 4.x\")\n}\n```",
	}
	rc, err := CompilePlanInputToRunContract(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rc.Setup) == 0 || rc.Setup[0].Name != "materialize issue repro file" {
		t.Fatalf("expected repro setup step, got %#v", rc.Setup)
	}
	if !strings.Contains(rc.Setup[0].Command, filepath.ToSlash("libbeat/common/kafka/zzz_repro_test.go")) {
		t.Fatalf("expected repro file path in setup command, got %#v", rc.Setup[0].Command)
	}
	if !strings.Contains(rc.Setup[0].Command, "func TestRepro_MajorVersionAliasUsesLatestMinor") {
		t.Fatalf("expected repro file contents in setup command, got %#v", rc.Setup[0].Command)
	}
}
