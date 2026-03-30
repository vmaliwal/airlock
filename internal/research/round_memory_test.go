package research

import (
	"strings"
	"testing"
)

func TestBuildNextRoundPlanInputIncludesPriorMemory(t *testing.T) {
	base := PlanInput{FailureText: "bug", Notes: "original"}
	prev := &AutofixSummary{Attempts: []AttemptOutcome{{
		Name:                   "wrong-fix",
		MutationKind:           "replace_line",
		Success:                false,
		ValidationFingerprints: []FailureFingerprint{{Signature: "test_failure:boom"}},
	}}}
	next := BuildNextRoundPlanInput(base, prev)
	if !strings.Contains(next.Notes, "failed_attempts: wrong-fix") {
		t.Fatalf("expected failed attempt memory, got %q", next.Notes)
	}
	if !strings.Contains(next.Notes, "failed_mutation_kinds: replace_line") {
		t.Fatalf("expected failed mutation kind memory, got %q", next.Notes)
	}
	if !strings.Contains(next.Notes, "failed_fingerprints: test_failure:boom") {
		t.Fatalf("expected failed fingerprint memory, got %q", next.Notes)
	}
}
