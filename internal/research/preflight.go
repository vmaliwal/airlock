package research

import "fmt"

type PreflightDecision struct {
	Profile              RepoProfile    `json:"profile"`
	Assessment           RepoAssessment `json:"assessment"`
	Route                string         `json:"route"`
	Reason               string         `json:"reason"`
	SuggestedCommands    []string       `json:"suggestedCommands,omitempty"`
	SuggestedVMBackend   string         `json:"suggestedVMBackend,omitempty"`
	SuggestedNextActions []string       `json:"suggestedNextActions,omitempty"`
}

func PreflightRepo(path string, vmBackend string) (PreflightDecision, error) {
	profile, err := DetectRepo(path)
	if err != nil {
		return PreflightDecision{}, err
	}
	assessment, err := AssessRepo(profile)
	if err != nil {
		return PreflightDecision{}, err
	}
	decision := PreflightDecision{Profile: profile, Assessment: assessment}
	switch assessment.Status {
	case "structurally_blocked":
		decision.Route = "stop"
		decision.Reason = "repo is structurally blocked; do not attempt mutation until source/bootstrap blockers are resolved"
		decision.SuggestedNextActions = []string{"inspect blockers", "repair bootstrap/source-of-truth", "rerun probe"}
	case "host_toolchain_blocked_vm_runnable":
		decision.Route = "vm"
		decision.Reason = "host toolchain is insufficient; route validation/mutation into a disposable VM"
		decision.SuggestedVMBackend = vmBackend
		decision.SuggestedCommands = []string{"airlock autofix-run <autofix.json>", "airlock attempt-run <attempt.json>", "airlock research-run <research.json>"}
		decision.SuggestedNextActions = []string{"use VM-backed run", "avoid host validation", "capture resulting artifacts and lessons"}
	default:
		decision.Route = "host"
		decision.Reason = "host execution is viable for bounded local workflows"
		decision.SuggestedCommands = []string{"airlock attempt-run <attempt.json>", "airlock autofix-run <autofix.json>", fmt.Sprintf("airlock probe %s", path)}
		decision.SuggestedNextActions = []string{"run bounded attempt/autofix locally", "promote to VM if trust or parity requires it"}
	}
	return decision, nil
}
