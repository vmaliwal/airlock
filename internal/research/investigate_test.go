package research

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInvestigateRepoIncludesPolicyAndHints(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-investigate-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "package.json"), []byte("{\"name\":\"example\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := InvestigateRepo(repo, "lima", false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Preflight.Route != "vm" {
		t.Fatalf("expected vm route, got %#v", report.Preflight)
	}
	if len(report.StrategyHints) == 0 {
		t.Fatalf("expected strategy hints, got %#v", report)
	}
	if report.HostExecutionPolicy["exceptionEnv"] != HostExecutionExceptionEnv {
		t.Fatalf("unexpected host policy info: %#v", report.HostExecutionPolicy)
	}
}
