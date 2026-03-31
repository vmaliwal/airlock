package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- Service-dependent detection ----

func TestIssueBodyServiceDependent(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"hmr reload", "vite HMR triggers full reload unexpectedly", true},
		{"dev server", "when running the dev server the module fails", true},
		{"stackblitz repro", "see reproduction at https://stackblitz.com/edit/xyz", true},
		{"open browser", "open a browser and click the button", true},
		{"program reload", "[vite] program reload", true},
		{"normal go test", "run go test ./pkg -run TestFoo", false},
		{"pytest", "pytest tests/ fails with AttributeError", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IssueBodyServiceDependent(tc.body); got != tc.want {
				t.Fatalf("IssueBodyServiceDependent(%q): want %v got %v", tc.body, tc.want, got)
			}
		})
	}
}

func TestIsSetupCommand(t *testing.T) {
	cases := []struct{ cmd string; want bool }{
		{"maturin develop --release", true},
		{"uv pip install pytest", true},
		{"pip install -r requirements.txt", true},
		{"npm install", true},
		{"pnpm install --frozen-lockfile", true},
		{"go build ./...", true},
		{"cargo build --release", true},
		{"go test ./pkg/foo -run TestBar", false},
		{"pytest tests/", false},
		{"node repro.js", false},
	}
	for _, tc := range cases {
		if got := IsSetupCommand(tc.cmd); got != tc.want {
			t.Fatalf("IsSetupCommand(%q): want %v got %v", tc.cmd, tc.want, got)
		}
	}
}

func TestClassifyIssueSignals_ServiceDependent(t *testing.T) {
	issue := GitHubIssue{Body: "Steps: start dev server, observe HMR failure on reload"}
	sig := ClassifyIssueSignals(issue, "node repro.js")
	if !sig.ServiceDependent {
		t.Fatal("expected service_dependent signal")
	}
	if sig.InferredCommandKind != "repro" {
		t.Fatalf("expected repro command kind, got %q", sig.InferredCommandKind)
	}
}

func TestClassifyIssueSignals_SetupCommand(t *testing.T) {
	issue := GitHubIssue{Body: "Run maturin develop then pytest"}
	sig := ClassifyIssueSignals(issue, "maturin develop --release")
	if sig.InferredCommandKind != "setup" {
		t.Fatalf("expected setup command kind, got %q", sig.InferredCommandKind)
	}
}

func TestClassifyIssueSignals_NoCommand(t *testing.T) {
	sig := ClassifyIssueSignals(GitHubIssue{Body: "inspect the code"}, "")
	if sig.InferredCommandKind != "none" {
		t.Fatalf("expected none command kind, got %q", sig.InferredCommandKind)
	}
}

// ---- Go resource lifecycle synthesis ----

func TestResourceLifecycleSignal(t *testing.T) {
	cases := []struct{ text string; want bool }{
		{"file descriptor leak in WriteSnapshotToDir", true},
		{"fd leak detected", true},
		{"os.Create is never closed", true},
		{"missing defer close", true},
		{"expected V4_1_0_0, got V4_0_0_0", false},
		{"unclosed code block", false},
	}
	for _, tc := range cases {
		if got := resourceLifecycleSignal(strings.ToLower(tc.text)); got != tc.want {
			t.Fatalf("resourceLifecycleSignal(%q): want %v got %v", tc.text, tc.want, got)
		}
	}
}

func TestSynthesizeGoResourceLifecycleAttempts(t *testing.T) {
	repo := t.TempDir()
	// Create a Go file with the pattern: os.Create + return without defer close
	src := `package modsdir

import (
	"os"
	"path/filepath"
)

func (m Manifest) WriteSnapshotToDir(dir string) error {
	fn := filepath.Join(dir, "manifest.json")
	w, err := os.Create(fn)
	if err != nil {
		return err
	}
	return m.WriteSnapshot(w) // w is never closed
}
`
	if err := os.MkdirAll(filepath.Join(repo, "internal", "modsdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "internal", "modsdir", "manifest.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := RepoProfile{TargetPath: repo, RepoType: "go"}
	input := PlanInput{
		FailureText: "file descriptor leak in modsdir.WriteSnapshotToDir",
		Notes:       "os.Create is called and the file is never closed",
	}
	attempts := synthesizeGoAttempts(input, profile, "go test ./internal/modsdir/...")
	if len(attempts) == 0 {
		t.Fatal("expected at least one resource lifecycle attempt, got none")
	}
	found := false
	for _, a := range attempts {
		if strings.Contains(a.Name, "defer close") || strings.Contains(a.Name, "resource") {
			found = true
			if a.MutationKind != "insert_after" {
				t.Fatalf("expected insert_after mutation kind, got %q", a.MutationKind)
			}
		}
	}
	if !found {
		t.Fatalf("expected a defer-close resource lifecycle attempt, got %+v", attempts)
	}
}

// ---- Python type-guard synthesis ----

func TestPythonTypeGuardSignal(t *testing.T) {
	cases := []struct{ text string; want bool }{
		{"AttributeError: 'NoneType' has no attribute 'x'", true},
		{"typeerror: isinstance check failed", true},
		{"none check missing before isinstance", true},
		{"expected/got normalization mismatch", false},
		{"fd leak", false},
	}
	for _, tc := range cases {
		if got := pythonTypeGuardSignal(strings.ToLower(tc.text)); got != tc.want {
			t.Fatalf("pythonTypeGuardSignal(%q): want %v got %v", tc.text, tc.want, got)
		}
	}
}

func TestSynthesizePythonTypeGuardAttempts(t *testing.T) {
	repo := t.TempDir()
	src := `def process(value):
    if value is not None and isinstance(value, str):
        return value.strip()
    return None
`
	if err := os.WriteFile(filepath.Join(repo, "utils.py"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := RepoProfile{TargetPath: repo, RepoType: "python"}
	attempts := synthesizePythonTypeGuardAttempts(profile, "pytest")
	if len(attempts) == 0 {
		t.Fatal("expected at least one type guard attempt, got none")
	}
	if attempts[0].MutationKind != "search_replace" {
		t.Fatalf("expected search_replace mutation kind, got %q", attempts[0].MutationKind)
	}
}

// ---- Fingerprint extraction helpers ----

func TestCollectFingerprintHintsFromFailureText(t *testing.T) {
	// collectFingerprintHintsFromFailureText wraps ExtractFailureSignatures,
	// which looks for Go test failure patterns in the failure text.
	// For a plain prose description it returns nil; verify it doesn't panic
	// and returns a slice (may be empty).
	hints := collectFingerprintHintsFromFailureText("file descriptor leak WriteSnapshotToDir os.Create")
	// Result is nil or empty — that's correct; prose has no Go test signals.
	_ = hints

	// With a real Go test failure pattern it should extract a signature.
	goTestFailure := "--- FAIL: TestWriteSnapshotToDir (0.01s)"
	sigs := collectFingerprintHintsFromFailureText(goTestFailure)
	if len(sigs) == 0 {
		t.Fatalf("expected fingerprint hint from Go test failure line, got none")
	}
	found := false
	for _, s := range sigs {
		if strings.Contains(s, "TestWriteSnapshotToDir") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected TestWriteSnapshotToDir in hints, got %v", sigs)
	}
}
