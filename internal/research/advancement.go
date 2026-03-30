package research

import "strings"

type AdvancementDecision struct {
	ShouldAdvance           bool   `json:"should_advance"`
	CredibleAdvancement     bool   `json:"credible_advancement"`
	VerifiedIssueResolution bool   `json:"verified_issue_resolution"`
	Reason                  string `json:"reason"`
	FailureCategory         string `json:"failure_category,omitempty"`
	ReproStatus             string `json:"repro_status"`
	ValidationScope         string `json:"validation_scope,omitempty"`
	FixConfidence           string `json:"fix_confidence,omitempty"`
	RegressionDetected      bool   `json:"regression_detected"`
}

func DecideAdvancement(proof ProofState, fixApplied, targetValidated, regressionDetected bool) AdvancementDecision {
	decision := AdvancementDecision{
		ReproStatus:        proof.ReproStatus,
		ValidationScope:    proof.ValidationScope,
		FixConfidence:      proof.FixConfidence,
		RegressionDetected: regressionDetected,
	}
	if !fixApplied {
		decision.Reason = "no fix applied"
		decision.FailureCategory = "no_fix_applied"
		return decision
	}
	switch proof.ReproStatus {
	case ReproStatusInfraFailure:
		decision.Reason = "reproduction failed due to infrastructure/tooling failure"
		decision.FailureCategory = "infra_failure"
		return decision
	case ReproStatusBootstrapFailure:
		decision.Reason = "reproduction failed due to bootstrap/toolchain failure"
		decision.FailureCategory = "bootstrap_failure"
		return decision
	case ReproStatusEnvBlocked:
		decision.Reason = "reproduction blocked by missing environment or credentials"
		decision.FailureCategory = "env_blocked"
		return decision
	case ReproStatusNotReproduced:
		decision.Reason = "bug was not reproduced before mutation"
		decision.FailureCategory = "not_reproduced"
		return decision
	}
	if !targetValidated {
		decision.Reason = "post-fix validation did not pass"
		decision.FailureCategory = "validation_failed"
		return decision
	}
	if regressionDetected {
		decision.Reason = "regression detected during broader validation"
		decision.FailureCategory = "regression_detected"
		return decision
	}
	if !isCredibleFixConfidence(proof.FixConfidence) {
		decision.Reason = "fix confidence is too weak to advance"
		decision.FailureCategory = "weak_evidence"
		return decision
	}
	decision.ShouldAdvance = true
	decision.CredibleAdvancement = true
	decision.VerifiedIssueResolution = true
	decision.Reason = "reproduced before mutation and passed post-fix validation with sufficient confidence"
	return decision
}

func isCredibleFixConfidence(conf string) bool {
	switch strings.ToLower(strings.TrimSpace(conf)) {
	case "medium", "high":
		return true
	default:
		return false
	}
}
