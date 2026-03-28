package research

import base "github.com/vmaliwal/airlock/internal/contract"

type ExecutionPolicyDecision struct {
	Preflight                  PreflightDecision `json:"preflight"`
	BackendKind                base.BackendKind  `json:"backendKind,omitempty"`
	HostExecutionExceptionUsed bool              `json:"hostExecutionExceptionUsed"`
}

func DecideExecutionPolicy(repo string, vmBackend string, allowHostExecution bool) (ExecutionPolicyDecision, error) {
	decision, err := PreflightRepo(repo, vmBackend, allowHostExecution)
	if err != nil {
		return ExecutionPolicyDecision{}, err
	}
	return ExecutionPolicyDecision{
		Preflight:                  decision,
		BackendKind:                base.BackendKind(decision.SuggestedVMBackend),
		HostExecutionExceptionUsed: allowHostExecution && decision.Route == "host",
	}, nil
}
