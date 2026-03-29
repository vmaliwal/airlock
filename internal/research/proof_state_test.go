package research

import "testing"

func TestDeriveReadOnlyProofState(t *testing.T) {
	proof := deriveReadOnlyProofState(EvaluationResult{Passed: true})
	if proof.ReproStatus != "reproduced" || proof.ValidationScope != "reproduction_only" || proof.FixConfidence != "none" {
		t.Fatalf("unexpected proof state: %#v", proof)
	}
}

func TestDeriveMutateProofStateHighConfidence(t *testing.T) {
	proof := deriveMutateProofState(
		EvaluationResult{Passed: true},
		EvaluationResult{Passed: true},
		EvaluationResult{Passed: true},
		true,
		true,
		true,
		1,
		true,
	)
	if proof.FixConfidence != "high" || proof.ReproStatus != "reproduced" {
		t.Fatalf("unexpected proof state: %#v", proof)
	}
}

func TestDeriveMutateProofStateLowWithoutRepro(t *testing.T) {
	proof := deriveMutateProofState(
		EvaluationResult{Passed: false},
		EvaluationResult{Passed: true},
		EvaluationResult{Passed: true},
		true,
		true,
		false,
		0,
		false,
	)
	if proof.FixConfidence != "low" || proof.ReproStatus != "not_reproduced" {
		t.Fatalf("unexpected proof state: %#v", proof)
	}
}
