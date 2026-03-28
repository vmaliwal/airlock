package research

import (
	"os"
	"path/filepath"
	"testing"
)

func pint(v int) *int           { return &v }
func pfloat(v float64) *float64 { return &v }

func TestEvaluateRepeated(t *testing.T) {
	res := EvaluateRepeated([]CommandResult{{ExitCode: 1}, {ExitCode: 0}}, SuccessRule{ExitCode: pint(0), MinPassRate: pfloat(1.0)})
	if res.Passed {
		t.Fatal("expected repeated evaluation to fail")
	}
	if res.FailCount != 1 {
		t.Fatalf("expected 1 failure, got %d", res.FailCount)
	}
}

func TestSummarizeFailures(t *testing.T) {
	fps := SummarizeFailures([]CommandResult{{Command: "go test", ExitCode: 1, Stdout: "--- FAIL: TestOne\nFAIL\nFAIL\tgithub.com/x/y\t0.123s"}})
	if len(fps) == 0 {
		t.Fatal("expected fingerprints")
	}
	found := false
	for _, fp := range fps {
		if fp.Signature == "package_failure:github.com/x/y" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected package fingerprint, got %#v", fps)
	}
}

func TestDetectRepoGoNested(t *testing.T) {
	profile, err := DetectRepo("../../")
	if err != nil {
		t.Fatal(err)
	}
	if profile.RepoRoot == "" {
		t.Fatal("expected repo root")
	}
}

func TestAssessRepoBlocksEmptyLocalReplaceTargets(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-probe-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/test\n\ngo 1.24.0\n\nreplace example.com/lib => ./modules/lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "modules", "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	profile, err := DetectRepo(repo)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := AssessRepo(profile)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Runnable {
		t.Fatalf("expected blocked assessment, got %#v", assessment)
	}
}

func TestAssessRepoMonorepoRootNeedsConcreteTarget(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-monorepo-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "libs", "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "libs", "core", "pyproject.toml"), []byte("[project]\nname='core'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile, err := DetectRepo(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.DiscoveredTargets) == 0 {
		t.Fatalf("expected discovered targets, got %#v", profile)
	}
	assessment, err := AssessRepo(profile)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Status != "monorepo_target_required" || assessment.Runnable || assessment.VMRunnable {
		t.Fatalf("unexpected assessment: %#v", assessment)
	}
}

func TestCompareGoVersions(t *testing.T) {
	if compareGoVersions("1.21.3", "1.24.0") >= 0 {
		t.Fatal("expected 1.21.3 < 1.24.0")
	}
	if compareGoVersions("1.24.0", "1.24") != 0 {
		t.Fatal("expected version equality")
	}
	if compareGoVersions("1.25.1", "1.24.9") <= 0 {
		t.Fatal("expected 1.25.1 > 1.24.9")
	}
}

func TestAssessRepoBootstrapNeededNode(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-node-bootstrap-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte("{\"name\":\"example\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile, err := DetectRepo(repo)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := AssessRepo(profile)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Status != "bootstrap_needed_vm_preferred" {
		t.Fatalf("unexpected assessment: %#v", assessment)
	}
}

func TestAssessRepoPartialRunnableScope(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-partial-scope-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "apps", "web", "package.json"), []byte("{\"name\":\"web\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "apps", "web", "package-lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile, err := DetectRepo(filepath.Join(repo, "apps", "web"))
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := AssessRepo(profile)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Status != "partial_runnable_scope" {
		t.Fatalf("unexpected assessment: %#v", assessment)
	}
}
