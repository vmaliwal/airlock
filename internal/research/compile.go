package research

import (
	base64 "encoding/base64"
	"encoding/json"
	"strings"

	base "github.com/vmaliwal/airlock/internal/contract"
)

const researchRunnerTemplate = `/tmp/airlock-researchguest __CONTRACT_B64__`

func CompileRunContract(c RunContract) (base.Contract, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return base.Contract{}, err
	}
	stepCommand := strings.ReplaceAll(researchRunnerTemplate, "__CONTRACT_B64__", base64.StdEncoding.EncodeToString(payload))
	air := c.Airlock
	air.Steps = []base.Step{{Name: "research-runner", Run: stepCommand, TimeoutSeconds: 3600, AllowFailure: false}}
	return air, nil
}
