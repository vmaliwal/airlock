package research

import (
	"encoding/json"
	"fmt"

	base "github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/runner"
)

func ExecuteCompiledContract(c base.Contract) (string, error) {
	b, err := runner.NewBackend(c.Backend.Kind)
	if err != nil {
		return "", err
	}
	if errs := b.CheckPrereqs(); len(errs) > 0 {
		data, _ := json.MarshalIndent(errs, "", "  ")
		return "", fmt.Errorf(string(data))
	}
	result, err := b.Run(c)
	if err != nil {
		return "", err
	}
	return result.SummaryPath, nil
}
