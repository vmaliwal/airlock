package research

import "testing"

func TestDecideAdvancementCredibleAndVerified(t *testing.T) {
	decision := DecideAdvancement(ProofState{ReproStatus: ReproStatusReproduced, ValidationScope: "target+broader", FixConfidence: "high"}, true, true, false)
	if !decision.ShouldAdvance || !decision.CredibleAdvancement || !decision.VerifiedIssueResolution {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestDecideAdvancementRejectsBootstrapFailure(t *testing.T) {
	decision := DecideAdvancement(ProofState{ReproStatus: ReproStatusBootstrapFailure, ValidationScope: "target_only", FixConfidence: "medium"}, true, true, false)
	if decision.ShouldAdvance || decision.CredibleAdvancement || decision.VerifiedIssueResolution {
		t.Fatalf("unexpected decision: %#v", decision)
	}
	if decision.FailureCategory != "bootstrap_failure" {
		t.Fatalf("expected bootstrap failure category, got %#v", decision)
	}
}
