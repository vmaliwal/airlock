package research

import "testing"

func TestInferBareLineCommand(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"go test", "Some text\ngo test ./pkg/foo -run TestBar\nmore", "go test ./pkg/foo -run TestBar"},
		{"pytest bare", "run:\npytest tests/ -k slow", "pytest tests/ -k slow"},
		{"python -m pytest", "python -m pytest tests/unit", "python -m pytest tests/unit"},
		{"uv run pytest", "to reproduce:\nuv run pytest tests/", "uv run pytest tests/"},
		{"node repro", "$ node repro.js\n[error]", "node repro.js"},
		{"node mjs", "node runner.mjs --verbose", "node runner.mjs --verbose"},
		{"pnpm test", "pnpm test --reporter=dot", "pnpm test --reporter=dot"},
		{"npm test", "npm test -- --run", "npm test -- --run"},
		{"yarn test", "yarn test --filter pkg", "yarn test --filter pkg"},
		{"just target", "just test-unit", "just test-unit"},
		{"cargo test", "cargo test --lib", "cargo test --lib"},
		{"prompt stripped", "$ go test ./...", "go test ./..."},
		{"prompt stripped2", "% pytest tests/", "pytest tests/"},
		{"no match", "open a browser and click things", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferBareLineCommand(tc.body)
			if got != tc.want {
				t.Fatalf("inferBareLineCommand(%q)\n  want %q\n   got %q", tc.body, tc.want, got)
			}
		})
	}
}

func TestInferFencedBlockCommand(t *testing.T) {
	body := "Steps to reproduce:\n\n```sh\n$ uv run pytest tests/integration\n```\n"
	got := inferFencedBlockCommand(body)
	if got != "uv run pytest tests/integration" {
		t.Fatalf("want uv run pytest tests/integration, got %q", got)
	}
}

func TestInferFailingCommandFromIssue_FencedFallback(t *testing.T) {
	issue := GitHubIssue{
		Body: "### Reproduction\n\nRun:\n\n```bash\nnode repro.js\n```\n\nExpected: no error\n",
	}
	got := inferFailingCommandFromIssue(issue)
	if got != "node repro.js" {
		t.Fatalf("want 'node repro.js', got %q", got)
	}
}

func TestInferFailingCommandFromIssue_UVRunInFenced(t *testing.T) {
	issue := GitHubIssue{
		Body: "to reproduce the bug:\n\n```sh\nuv run pytest tests/\n```\n",
	}
	got := inferFailingCommandFromIssue(issue)
	if got != "uv run pytest tests/" {
		t.Fatalf("want 'uv run pytest tests/', got %q", got)
	}
}

func TestInferFailingCommandFromIssue_NoMatch(t *testing.T) {
	issue := GitHubIssue{
		Body: "Open a browser and reproduce by clicking the button. No test command provided.",
	}
	got := inferFailingCommandFromIssue(issue)
	if got != "" {
		t.Fatalf("expected no command, got %q", got)
	}
}
