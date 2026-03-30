package research

import "testing"

func TestDeriveReadOnlyProofState(t *testing.T) {
	proof := deriveReadOnlyProofState([]CommandResult{{Command: "go test ./pkg", ExitCode: 1, Stderr: "expected failure from issue repro"}}, EvaluationResult{Passed: true})
	if proof.ReproStatus != ReproStatusReproduced || proof.ValidationScope != "reproduction_only" || proof.FixConfidence != "none" {
		t.Fatalf("unexpected proof state: %#v", proof)
	}
}

func TestDeriveReadOnlyProofStateBootstrapFailure(t *testing.T) {
	proof := deriveReadOnlyProofState([]CommandResult{{Command: "go test ./pkg", ExitCode: 1, Stderr: "bash: line 1: go: command not found"}}, EvaluationResult{Passed: true})
	if proof.ReproStatus != ReproStatusBootstrapFailure {
		t.Fatalf("expected bootstrap failure, got %#v", proof)
	}
}

func TestDeriveMutateProofStateHighConfidence(t *testing.T) {
	proof := deriveMutateProofState(
		[]CommandResult{{Command: "pytest -q", ExitCode: 1, Stderr: "assert 1 == 2"}},
		EvaluationResult{Passed: true},
		EvaluationResult{Passed: true},
		EvaluationResult{Passed: true},
		true,
		true,
		true,
		1,
		true,
	)
	if proof.FixConfidence != "high" || proof.ReproStatus != ReproStatusReproduced {
		t.Fatalf("unexpected proof state: %#v", proof)
	}
}

func TestDeriveMutateProofStateLowWithoutRepro(t *testing.T) {
	proof := deriveMutateProofState(
		nil,
		EvaluationResult{Passed: false},
		EvaluationResult{Passed: true},
		EvaluationResult{Passed: true},
		true,
		true,
		false,
		0,
		false,
	)
	if proof.FixConfidence != "low" || proof.ReproStatus != ReproStatusNotReproduced {
		t.Fatalf("unexpected proof state: %#v", proof)
	}
}
