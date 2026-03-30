package research

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDraftPRArtifact(t *testing.T) {
	tmp := t.TempDir()
	result := FixResult{
		Issue: GitHubIssue{
			Owner:   "owner",
			Repo:    "repo",
			Number:  123,
			Title:   "fix parser edge case",
			HTMLURL: "https://github.com/owner/repo/issues/123",
		},
		RepoPath:               tmp,
		PlanInput:              PlanInput{FailingCommand: "go test ./pkg/foo -run TestEdgeCase", FailureText: "fix parser edge case"},
		ReviewPacketPath:       filepath.Join(tmp, "review-packet.md"),
		ReadonlySummaryPath:    filepath.Join(tmp, "readonly-summary.json"),
		AutofixContractSummary: filepath.Join(tmp, "autofix-summary.json"),
		Synthesis:              SynthesisReport{Summary: "planner suggested preserving buffered content at EOF"},
	}
	summary := RunSummary{
		ReproStatus:             ReproStatusReproduced,
		ValidationScope:         "target_only",
		FixConfidence:           "medium",
		CredibleAdvancement:     true,
		VerifiedIssueResolution: false,
	}
	proof := ProofState{ReproStatus: ReproStatusReproduced, ValidationScope: "target_only", FixConfidence: "medium", ConfidenceReason: "reproduced before patch and passed target validation"}
	path, err := WriteDraftPRArtifact(result, summary, proof)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	checks := []string{
		"fix parser edge case",
		"## Summary",
		"Issue: https://github.com/owner/repo/issues/123",
		"Reproduction command: `go test ./pkg/foo -run TestEdgeCase`",
		"planner suggested preserving buffered content at EOF",
		"Review packet: `" + filepath.Join(tmp, "review-packet.md") + "`",
		"Full issue-resolution proof is not yet complete.",
	}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("draft pr missing %q\n%s", want, text)
		}
	}
}
