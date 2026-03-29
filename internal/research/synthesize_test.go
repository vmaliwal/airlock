package research

import (
	"path/filepath"
	"testing"

	"github.com/vmaliwal/airlock/internal/util"
)

func TestSynthesizeAutofixPlanUnclosedCodeBlock(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"libs/text-splitters/pyproject.toml": "[project]\nname='text-splitters'\n",
		"libs/text-splitters/uv.lock":        "version = 1\n",
		"libs/text-splitters/markdown.py":    "def _resolve_code_chunk(self, current_line, raw_lines):\n    chunk = current_line\n    while raw_lines:\n        raw_line = raw_lines.pop(0)\n        chunk += raw_line\n        if self._match_code(raw_line):\n            return chunk\n    return \"\"\n",
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
		Notes:          "_resolve_code_chunk returns empty string instead of accumulated chunk when closing fence is missing",
	}
	report, err := SynthesizeAutofixPlan(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Supported || report.AutofixPlan == nil || len(report.Attempts) == 0 {
		t.Fatalf("expected synthesized attempts, got %#v", report)
	}
	m := report.Attempts[0].Attempt.Mutation.ReplaceLine
	if m == nil || m.NewLine != "        return chunk" {
		t.Fatalf("expected replace-line eof attempt, got %#v", report.Attempts[0])
	}
}

func TestSynthesizeAutofixPlanGoExpectedGotNormalization(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"go.mod":          "module example.com/beats\n\ngo 1.23\n",
		"version.go":      "package beats\n\nfunc alias() string {\n\treturn \"V4_0_0_0\"\n}\n",
		"version_test.go": "package beats\n",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/beats.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	input := PlanInput{RepoPath: repo, FailureText: "expected V4_1_0_0, got V4_0_0_0", FailingCommand: "go test ./... -run TestKafkaAlias -count=1"}
	report, err := SynthesizeAutofixPlan(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Supported || len(report.Attempts) == 0 {
		t.Fatalf("expected synthesized go attempt, got %#v", report)
	}
	m := report.Attempts[0].Attempt.Mutation.ReplaceLine
	if m == nil || m.NewLine != "\treturn \"V4_1_0_0\"" {
		t.Fatalf("expected go replace-line attempt, got %#v", report.Attempts[0])
	}
}

func TestSynthesizeAutofixPlanEmptyReasoningContent(t *testing.T) {
	repo := t.TempDir()
	if err := InitTempGitRepo(repo, map[string]string{
		"libs/core/pyproject.toml": "[project]\nname='core'\n",
		"libs/core/uv.lock":        "version = 1\n",
		"libs/core/messages.py":    "reasoning_content = additional_kwargs.get(\"reasoning_content\")\nif reasoning_content is not None and isinstance(reasoning_content, str):\n    return {\"type\": \"reasoning\", \"reasoning\": reasoning_content}\n",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := util.RunLocal("git", []string{"remote", "add", "origin", "git@github.com:example/langchain.git"}, util.RunOptions{Cwd: repo}); err != nil {
		t.Fatal(err)
	}
	input := PlanInput{
		RepoPath:       filepath.Join(repo, "libs", "core"),
		IssueURL:       "https://github.com/langchain-ai/langchain/issues/36194",
		FailingCommand: "python -m pytest tests -k empty_string",
		FailureText:    "_extract_reasoning_from_additional_kwargs should ignore empty strings in reasoning_content",
	}
	report, err := SynthesizeAutofixPlan(input, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Supported || report.AutofixPlan == nil || len(report.Attempts) == 0 {
		t.Fatalf("expected synthesized attempts, got %#v", report)
	}
	m := report.Attempts[0].Attempt.Mutation.SearchReplace
	if m == nil || m.NewText != "if isinstance(reasoning_content, str) and reasoning_content:" {
		t.Fatalf("expected search-replace empty-string attempt, got %#v", report.Attempts[0])
	}
}
