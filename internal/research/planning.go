package research

import (
	"os"
	"path/filepath"
	"strings"
)

const LessonsRootEnv = "AIRLOCK_LESSONS_ROOT"

type MutationKindScore struct {
	Kind    string   `json:"kind"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
}

type PlanReport struct {
	Input                PlanInput           `json:"input"`
	Investigation        InvestigationReport `json:"investigation"`
	RankedMutationKinds  []MutationKindScore `json:"rankedMutationKinds,omitempty"`
	CandidateActionKinds []string            `json:"candidateActionKinds,omitempty"`
	LessonsSearchRoots   []string            `json:"lessonsSearchRoots,omitempty"`
	CandidateCommands    []string            `json:"candidateCommands,omitempty"`
}

func PlanRepo(path string, vmBackend string, allowHostExecution bool) (PlanReport, error) {
	return PlanFromInput(PlanInput{RepoPath: path}, vmBackend, allowHostExecution)
}

func PlanFromInput(input PlanInput, vmBackend string, allowHostExecution bool) (PlanReport, error) {
	investigation, err := InvestigateRepo(input.RepoPath, vmBackend, allowHostExecution)
	if err != nil {
		return PlanReport{}, err
	}
	if looksFlakyOrHang(input.FailureText) {
		investigation.Assessment.Warnings = dedupeStrings(append(investigation.Assessment.Warnings, "flaky_candidate"))
		investigation.StrategyHints = dedupeStrings(append(investigation.StrategyHints, "failure text suggests a flaky/hang class; prefer bounded reruns, timeouts, and stability-oriented validation"))
	}
	roots := lessonSearchRoots(investigation.Profile.RepoRoot)
	lessons := loadLessonsFromRoots(roots)
	fingerprintHints := collectFingerprintHintsFromFailureText(input.FailureText)
	ranked := rankMutationKindsWithContext(investigation.Profile, lessons, fingerprintHints)
	actionKinds := candidateActionKinds(ranked)
	candidateCommands := rankedCommands(input, investigation)
	return PlanReport{
		Input:                input,
		Investigation:        investigation,
		RankedMutationKinds:  ranked,
		CandidateActionKinds: actionKinds,
		LessonsSearchRoots:   roots,
		CandidateCommands:    candidateCommands,
	}, nil
}

func lessonSearchRoots(repoRoot string) []string {
	roots := []string{}
	if v := os.Getenv(LessonsRootEnv); v != "" {
		roots = append(roots, v)
	}
	if repoRoot != "" {
		roots = append(roots, repoRoot)
		roots = append(roots, filepath.Dir(repoRoot))
	}
	return dedupeStrings(roots)
}

func defaultMutationKinds(repoType string) []string {
	switch repoType {
	case "go":
		return []string{"replace_line", "search_replace", "insert_after", "apply_patch"}
	case "python":
		return []string{"search_replace", "replace_line", "create_file", "apply_patch"}
	case "node":
		return []string{"search_replace", "replace_line", "insert_after", "apply_patch"}
	default:
		return []string{"search_replace", "replace_line", "apply_patch"}
	}
}

func candidateActionKinds(ranked []MutationKindScore) []string {
	out := []string{}
	for i, item := range ranked {
		if i >= 4 {
			break
		}
		switch item.Kind {
		case "apply_patch", "search_replace", "replace_line", "insert_after", "create_file":
			out = append(out, item.Kind)
		}
	}
	return dedupeStrings(out)
}

func rankedCommands(input PlanInput, investigation InvestigationReport) []string {
	out := []string{}
	if input.FailingCommand != "" {
		out = append(out, input.FailingCommand)
	}
	out = append(out, investigation.CandidateReproduction...)
	out = append(out, investigation.CandidateValidation...)
	if input.IssueURL != "" {
		out = append(out, "issue_context="+input.IssueURL)
	}
	if input.FailureText != "" {
		out = append(out, "failure_text_present")
	}
	return dedupeStrings(out)
}

func looksFlakyOrHang(s string) bool {
	s = strings.ToLower(s)
	for _, needle := range []string{"timeout", "timed out", "hang", "flaky", "intermittent", "cancelled", "canceled"} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
