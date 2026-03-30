package research

import (
	"path/filepath"
	"testing"
)

func TestAppendAndLoadRunSummary(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(MetricsDirEnv, dir)
	summary := RunSummary{RunID: "run-1", CustomerID: "acme", RepoKey: "elastic/beats", Entrypoint: "fix", CredibleAdvancement: true, VerifiedIssueResolution: true}
	if err := AppendRunSummary(summary); err != nil {
		t.Fatal(err)
	}
	items, err := LoadRunSummaries(filepath.Join(dir, "runs.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RunID != "run-1" {
		t.Fatalf("unexpected run summaries: %#v", items)
	}
}

func TestSummarizeRunMetrics(t *testing.T) {
	items := []RunSummary{
		{RunID: "1", CustomerID: "acme", RepoKey: "elastic/beats", CredibleAdvancement: true, VerifiedIssueResolution: true},
		{RunID: "2", CustomerID: "acme", RepoKey: "elastic/beats"},
		{RunID: "3", CustomerID: "globex", RepoKey: "charmbracelet/gum", CredibleAdvancement: true},
	}
	summary := SummarizeRunMetrics(items)
	if summary.Global.Runs != 3 {
		t.Fatalf("expected 3 runs, got %#v", summary.Global)
	}
	if summary.Global.CredibleAdvancementRate != 2.0/3.0 {
		t.Fatalf("unexpected credible advancement rate: %#v", summary.Global)
	}
	if summary.Global.VerifiedIssueResolutionRate != 1.0/3.0 {
		t.Fatalf("unexpected verified issue resolution rate: %#v", summary.Global)
	}
	if summary.ByRepo["elastic/beats"].Runs != 2 {
		t.Fatalf("unexpected repo rollup: %#v", summary.ByRepo)
	}
	if summary.ByCustomer["acme"].Runs != 2 {
		t.Fatalf("unexpected customer rollup: %#v", summary.ByCustomer)
	}
}

func TestNormalizeRepoKey(t *testing.T) {
	if got := NormalizeRepoKey("git@github.com:vmaliwal/airlock.git"); got != "vmaliwal/airlock" {
		t.Fatalf("unexpected repo key: %s", got)
	}
}
