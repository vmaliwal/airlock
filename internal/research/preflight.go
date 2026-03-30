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

func PreflightRepo(path string, vmBackend string, allowHostExecution bool) (PreflightDecision, error) {
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
	case "monorepo_target_required":
		decision.Route = "stop"
		decision.Reason = "repo root is a monorepo entrypoint; choose a concrete package/module target before running attempts"
		decision.SuggestedNextActions = append(decision.SuggestedNextActions, assessment.Evidence...)
	case "host_toolchain_blocked_vm_runnable", "bootstrap_needed_vm_preferred", "partial_runnable_scope":
		decision.Route = "vm"
		decision.Reason = "repo should be executed in a disposable VM before mutation/validation proceeds"
		decision.SuggestedVMBackend = vmBackend
		decision.SuggestedCommands = []string{"airlock investigate <repo-path>", "airlock plan <repo-path>", "airlock autofix-run <autofix.json>", "airlock attempt-run <attempt.json>", "airlock research-run <research.json>"}
		decision.SuggestedNextActions = []string{"use VM-backed run", "avoid host validation", "capture resulting artifacts and lessons"}
	case "env_config_blocked":
		decision.Route = "stop"
		decision.Reason = "repo execution context is still underspecified; gather missing environment/bootstrap context before mutation"
		decision.SuggestedNextActions = append(decision.SuggestedNextActions, assessment.Evidence...)
	default:
		if !allowHostExecution {
			if vmBackend != "" {
				decision.Route = "vm"
				decision.Reason = "host execution is blocked by policy for unknown repo code; route execution into a disposable VM unless an explicit host exception is declared"
				decision.SuggestedVMBackend = vmBackend
				decision.SuggestedCommands = []string{"airlock investigate <repo-path>", "airlock autofix-run <autofix.json>", "airlock attempt-run <attempt.json>"}
				decision.SuggestedNextActions = []string{"use VM-backed execution", fmt.Sprintf("declare %s=1 only for an explicit host exception", HostExecutionExceptionEnv)}
			} else {
				decision.Route = "stop"
				decision.Reason = "host execution is blocked by policy for unknown repo code and no VM backend is ready"
				decision.SuggestedNextActions = []string{"run 'airlock check'", "configure a VM backend", fmt.Sprintf("declare %s=1 only for an explicit host exception", HostExecutionExceptionEnv)}
			}
		} else {
			decision.Route = "host"
			decision.Reason = "host execution exception was explicitly declared"
			decision.SuggestedCommands = []string{"airlock attempt-run <attempt.json>", "airlock autofix-run <autofix.json>", fmt.Sprintf("airlock probe %s", path)}
			decision.SuggestedNextActions = []string{"run bounded attempt/autofix locally", "prefer VM execution when trust or parity is uncertain"}
		}
	}
	return decision, nil
}
