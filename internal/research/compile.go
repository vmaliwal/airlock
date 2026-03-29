package research

import (
	base64 "encoding/base64"
	"encoding/json"
	"strings"

	base "github.com/vmaliwal/airlock/internal/contract"
)

const researchRunnerTemplate = `/tmp/airlock-researchguest __CONTRACT_B64__`

func CompileRunContract(c RunContract) (base.Contract, error) {
	rc := c
	if rc.Plan == nil {
		if localTarget := rc.LocalPlanningTargetPath(); localTarget != "" {
			plan, err := PlanFromInput(PlanInput{RepoPath: localTarget, Notes: rc.Objective}, string(rc.Airlock.Backend.Kind), rc.HostExecutionException)
			if err == nil {
				concrete := BuildConcretePlan(plan)
				rc.Plan = &concrete
			}
		}
	}
	if subdir := rc.Airlock.Repo.Subdir; subdir != "" {
		rc.Safety.AllowedPaths = prefixPaths(subdir, rc.Safety.AllowedPaths)
		rc.Safety.ForbiddenPaths = prefixPaths(subdir, rc.Safety.ForbiddenPaths)
	}
	payload, err := json.Marshal(rc)
	if err != nil {
		return base.Contract{}, err
	}
	stepCommand := strings.ReplaceAll(researchRunnerTemplate, "__CONTRACT_B64__", base64.StdEncoding.EncodeToString(payload))
	air := rc.Airlock
	air.Steps = []base.Step{{Name: "research-runner", Run: stepCommand, TimeoutSeconds: 3600, AllowFailure: false}}
	return air, nil
}
