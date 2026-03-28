package research

type PlanStep struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
	Path    string `json:"path,omitempty"`
}

type ConcretePlan struct {
	Input                PlanInput           `json:"input"`
	TargetRepo           string              `json:"targetRepo"`
	CandidateTargetPath  string              `json:"candidateTargetPath,omitempty"`
	ReproductionCommands []string            `json:"reproductionCommands,omitempty"`
	ValidationCommands   []string            `json:"validationCommands,omitempty"`
	MutationKinds        []MutationKindScore `json:"mutationKinds,omitempty"`
	Steps                []PlanStep          `json:"steps,omitempty"`
}

func BuildConcretePlan(report PlanReport) ConcretePlan {
	steps := []PlanStep{}
	for _, cmd := range report.Investigation.CandidateReproduction {
		steps = append(steps, PlanStep{Kind: "reproduction", Name: "candidate reproduction", Command: cmd})
	}
	for _, cmd := range report.Investigation.CandidateValidation {
		steps = append(steps, PlanStep{Kind: "validation", Name: "candidate validation", Command: cmd})
	}
	for _, kind := range report.RankedMutationKinds {
		steps = append(steps, PlanStep{Kind: "mutation_family", Name: kind.Kind})
	}
	return ConcretePlan{
		Input:                report.Input,
		TargetRepo:           report.Investigation.Profile.RepoRoot,
		CandidateTargetPath:  report.Investigation.Profile.TargetPath,
		ReproductionCommands: append([]string{}, report.Investigation.CandidateReproduction...),
		ValidationCommands:   append([]string{}, report.Investigation.CandidateValidation...),
		MutationKinds:        append([]MutationKindScore{}, report.RankedMutationKinds...),
		Steps:                steps,
	}
}
