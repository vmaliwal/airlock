package research

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vmaliwal/airlock/internal/util"
)

func TestCompilePlanInputToRunContractGoPrefixesToolchainBootstrap(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"go.mod":          "module example.com/beats\n\ngo 1.25.8\n",
		"version.go":      "package beats\n",
		"version_test.go": "package beats\n",
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
	}
	rc, err := CompilePlanInputToRunContract(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rc.Reproduction.Command, "go.dev/dl/go1.25.8.linux-") {
		t.Fatalf("expected go toolchain bootstrap in reproduction command, got %#v", rc.Reproduction.Command)
	}
	if !strings.Contains(rc.Reproduction.Command, "export PATH=/tmp/airlock-go/go/bin:$PATH") {
		t.Fatalf("expected PATH bootstrap in reproduction command, got %#v", rc.Reproduction.Command)
	}
	if len(rc.Airlock.Security.BootstrapAptPackages) == 0 || !contains(rc.Airlock.Security.BootstrapAptPackages, "curl") {
		t.Fatalf("expected curl bootstrap package for go repo, got %#v", rc.Airlock.Security.BootstrapAptPackages)
	}
	if !contains(rc.Airlock.Security.AllowedEnv, "GITHUB_TOKEN") {
		t.Fatalf("expected github token allowlist for github clone url, got %#v", rc.Airlock.Security.AllowedEnv)
	}
}

func TestCompilePlanInputToRunContractPythonSubdir(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"libs/text-splitters/pyproject.toml": "[project]\nname='text-splitters'\n",
		"libs/text-splitters/uv.lock":        "version = 1\n",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/langchain.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	input := PlanInput{
		RepoPath:       filepath.Join(repo, "libs", "text-splitters"),
		IssueURL:       "https://github.com/langchain-ai/langchain/issues/36186",
		FailingCommand: "python -m pytest tests -k unclosed_code_block",
		FailureText:    "ExperimentalMarkdownSyntaxTextSplitter silently discards content from unclosed code blocks",
	}
	rc, err := CompilePlanInputToRunContract(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if rc.Mode != "read_only" {
		t.Fatalf("expected read_only mode, got %#v", rc)
	}
	expectedCmd := ".venv/bin/python -m pytest tests -k unclosed_code_block"
	if rc.Reproduction.Command != expectedCmd || rc.Validation.TargetCommand != expectedCmd {
		t.Fatalf("expected venv-aware python command, got %#v", rc)
	}
	if len(rc.Setup) == 0 || rc.Setup[0].Name != "bootstrap python venv" {
		t.Fatalf("expected python bootstrap setup, got %#v", rc.Setup)
	}
	if !strings.Contains(rc.Setup[0].Command, ".venv/bin/python -m pip install -q -e .") {
		t.Fatalf("expected editable-install bootstrap policy, got %#v", rc.Setup)
	}
	if rc.Airlock.Repo.CloneURL != "https://github.com/example/langchain.git" {
		t.Fatalf("unexpected clone url: %#v", rc.Airlock.Repo)
	}
	if rc.Airlock.Repo.Subdir != "libs/text-splitters" {
		t.Fatalf("unexpected subdir: %#v", rc.Airlock.Repo)
	}
	if len(rc.Airlock.Security.BootstrapAptPackages) == 0 || len(rc.Airlock.Security.AllowHosts) == 0 {
		t.Fatalf("expected bootstrap/network defaults, got %#v", rc.Airlock.Security)
	}
}
