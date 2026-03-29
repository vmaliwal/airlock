package research

type ProofState struct {
	ReproStatus      string `json:"repro_status"`
	ValidationScope  string `json:"validation_scope"`
	FixConfidence    string `json:"fix_confidence"`
	ConfidenceReason string `json:"confidence_reason"`
}

func deriveReadOnlyProofState(repro EvaluationResult) ProofState {
	status := "not_reproduced"
	if repro.Passed {
		status = "reproduced"
	}
	return ProofState{
		ReproStatus:      status,
		ValidationScope:  "reproduction_only",
		FixConfidence:    "none",
		ConfidenceReason: "read-only run does not apply a fix; proof is limited to reproduction status",
	}
}

func deriveMutateProofState(repro, validation, neighbor EvaluationResult, broaderPassed, campaignPassed bool, hasNeighbor bool, broaderCount int, hasCampaign bool) ProofState {
	status := "not_reproduced"
	if repro.Passed {
		status = "reproduced"
	}
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
	if !repro.Passed {
		return ProofState{
			ReproStatus:      status,
			ValidationScope:  scope,
			FixConfidence:    "low",
			ConfidenceReason: "target bug was not reproduced before mutation, so fix proof is weak",
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
