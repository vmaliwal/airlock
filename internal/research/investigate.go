package research

import "fmt"

type InvestigationReport struct {
	Profile               RepoProfile       `json:"profile"`
	Assessment            RepoAssessment    `json:"assessment"`
	Preflight             PreflightDecision `json:"preflight"`
	CandidateReproduction []string          `json:"candidateReproduction,omitempty"`
	CandidateValidation   []string          `json:"candidateValidation,omitempty"`
	StrategyHints         []string          `json:"strategyHints,omitempty"`
	SuggestedNextActions  []string          `json:"suggestedNextActions,omitempty"`
	HostExecutionPolicy   map[string]any    `json:"hostExecutionPolicy"`
}

func InvestigateRepo(path string, vmBackend string, allowHostExecution bool) (InvestigationReport, error) {
	profile, err := DetectRepo(path)
	if err != nil {
		return InvestigationReport{}, err
	}
	assessment, err := AssessRepo(profile)
	if err != nil {
		return InvestigationReport{}, err
	}
	preflight, err := PreflightRepo(path, vmBackend, allowHostExecution)
	if err != nil {
		return InvestigationReport{}, err
	}
	repro := append([]string{}, profile.BaselineCommands...)
	validation := append([]string{}, profile.BaselineCommands...)
	if len(validation) == 0 && profile.TargetPath != "" {
		validation = append(validation, fmt.Sprintf("airlock probe %s", profile.TargetPath))
	}
	hints := strategyHintsFor(profile, assessment)
	next := append([]string{}, preflight.SuggestedNextActions...)
	if len(repro) > 0 {
		next = append(next, "narrow one reproduction candidate before mutation")
	}
	return InvestigationReport{
		Profile:               profile,
		Assessment:            assessment,
		Preflight:             preflight,
		CandidateReproduction: repro,
		CandidateValidation:   validation,
		StrategyHints:         hints,
		SuggestedNextActions:  dedupeStrings(next),
		HostExecutionPolicy: map[string]any{
			"exceptionDeclared": allowHostExecution,
			"exceptionEnv":      HostExecutionExceptionEnv,
		},
	}, nil
}

func strategyHintsFor(profile RepoProfile, assessment RepoAssessment) []string {
	hints := []string{}
	switch profile.RepoType {
	case "go":
		hints = append(hints,
			"prefer narrowed go test reproduction before broad package validation",
			"prefer small git-diff-safe fixes in the target package before broader mutation",
		)
	case "python":
		hints = append(hints,
			"prefer venv-first bootstrap in guest environments",
			"prefer direct python reproduction when repo harness imports create unrelated blockers",
		)
	case "node":
		hints = append(hints,
			"prefer bounded command reproduction and explicit timeout handling for CLI issues",
			"watch for dependency/bootstrap requirements before blaming runtime behavior",
		)
	}
	if assessment.Status == "monorepo_target_required" {
		hints = append(hints, "choose a concrete package/module target before mutation")
	}
	if assessment.Status == "structurally_blocked" {
		hints = append(hints, "do not mutate until bootstrap/source blockers are resolved")
	}
	return dedupeStrings(hints)
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
