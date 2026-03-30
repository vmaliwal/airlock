package research

import "strings"

const (
	ReproStatusReproduced       = "reproduced"
	ReproStatusNotReproduced    = "not_reproduced"
	ReproStatusInfraFailure     = "infra_failure"
	ReproStatusBootstrapFailure = "bootstrap_failure"
	ReproStatusEnvBlocked       = "env_blocked"
)

type ProofState struct {
	ReproStatus      string `json:"repro_status"`
	ValidationScope  string `json:"validation_scope"`
	FixConfidence    string `json:"fix_confidence"`
	ConfidenceReason string `json:"confidence_reason"`
}

func deriveReadOnlyProofState(runs []CommandResult, repro EvaluationResult) ProofState {
	status := deriveReproStatus(runs, repro)
	reason := "read-only run does not apply a fix; proof is limited to reproduction status"
	switch status {
	case ReproStatusInfraFailure:
		reason = "read-only run hit infrastructure/tooling failure; bug reproduction is not established"
	case ReproStatusBootstrapFailure:
		reason = "read-only run hit bootstrap/toolchain failure; bug reproduction is not established"
	case ReproStatusEnvBlocked:
		reason = "read-only run was blocked by missing environment or credentials; bug reproduction is not established"
	}
	return ProofState{
		ReproStatus:      status,
		ValidationScope:  "reproduction_only",
		FixConfidence:    "none",
		ConfidenceReason: reason,
	}
}

func deriveMutateProofState(reproRuns []CommandResult, repro, validation, neighbor EvaluationResult, broaderPassed, campaignPassed bool, hasNeighbor bool, broaderCount int, hasCampaign bool) ProofState {
	status := deriveReproStatus(reproRuns, repro)
	scope := "target_only"
	if hasNeighbor && broaderCount > 0 {
		scope = "target+neighbor+broader"
	} else if hasNeighbor {
		scope = "target+neighbor"
	} else if broaderCount > 0 {
		scope = "target+broader"
	}
	if hasCampaign {
		scope += "+campaign"
	}
	if status != ReproStatusReproduced {
		return ProofState{
			ReproStatus:      status,
			ValidationScope:  scope,
			FixConfidence:    "low",
			ConfidenceReason: reproStatusReason(status),
		}
	}
	if !(validation.Passed && neighbor.Passed && broaderPassed && campaignPassed) {
		return ProofState{
			ReproStatus:      status,
			ValidationScope:  scope,
			FixConfidence:    "low",
			ConfidenceReason: "reproduction succeeded, but post-patch validation did not fully pass",
		}
	}
	if hasNeighbor || broaderCount > 0 || hasCampaign {
		return ProofState{
			ReproStatus:      status,
			ValidationScope:  scope,
			FixConfidence:    "high",
			ConfidenceReason: "reproduced before patch and passed target plus additional post-patch validation",
		}
	}
	return ProofState{
		ReproStatus:      status,
		ValidationScope:  scope,
		FixConfidence:    "medium",
		ConfidenceReason: "reproduced before patch and passed target validation, but additional validation scope was limited",
	}
}

func deriveReproStatus(runs []CommandResult, repro EvaluationResult) string {
	for _, run := range runs {
		text := strings.ToLower(strings.Join([]string{run.Command, run.Stdout, run.Stderr}, "\n"))
		if isEnvBlockedText(text) {
			return ReproStatusEnvBlocked
		}
		if isBootstrapFailureText(text) {
			return ReproStatusBootstrapFailure
		}
		if isInfraFailureText(text) {
			return ReproStatusInfraFailure
		}
	}
	if repro.Passed {
		return ReproStatusReproduced
	}
	return ReproStatusNotReproduced
}

func isEnvBlockedText(text string) bool {
	patterns := []string{
		"missing required env",
		"environment variable",
		"must set",
		"not set",
		"missing token",
		"api key",
		"credentials",
		"unauthorized",
		"insufficient scopes",
		"authentication failed",
	}
	return containsAny(text, patterns)
}

func isBootstrapFailureText(text string) bool {
	patterns := []string{
		"command not found",
		"modulenotfounderror",
		"no module named",
		"cannot find module",
		"importerror",
		"externally-managed-environment",
		"no such file or directory: '.venv",
		"npm err! enoent",
		"go: command not found",
		"pytest: command not found",
		"cargo: command not found",
	}
	return containsAny(text, patterns)
}

func isInfraFailureText(text string) bool {
	patterns := []string{
		"executable file not found",
		"failed to connect",
		"connection refused",
		"network is unreachable",
		"no such file or directory",
		"operation not permitted",
		"permission denied",
		"timed out",
	}
	return containsAny(text, patterns)
}

func containsAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func reproStatusReason(status string) string {
	switch status {
	case ReproStatusInfraFailure:
		return "reproduction was blocked by infrastructure/tooling failure, so fix proof is weak"
	case ReproStatusBootstrapFailure:
		return "reproduction was blocked by bootstrap/toolchain failure, so fix proof is weak"
	case ReproStatusEnvBlocked:
		return "reproduction was blocked by missing environment or credentials, so fix proof is weak"
	default:
		return "target bug was not reproduced before mutation, so fix proof is weak"
	}
}
