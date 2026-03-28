package research

import "testing"

func TestBuildConcretePlan(t *testing.T) {
	report := PlanReport{
		Input: PlanInput{RepoPath: "/tmp/repo"},
		Investigation: InvestigationReport{
			Profile:               RepoProfile{RepoRoot: "/tmp/repo", TargetPath: "/tmp/repo/pkg"},
			CandidateReproduction: []string{"go test ./pkg -run TestX"},
			CandidateValidation:   []string{"go test ./pkg/..."},
		},
		RankedMutationKinds: []MutationKindScore{{Kind: "nil_guard", Score: 10}},
	}
	plan := BuildConcretePlan(report)
	if plan.TargetRepo != "/tmp/repo" || len(plan.Steps) == 0 {
		t.Fatalf("unexpected concrete plan: %#v", plan)
	}
}
