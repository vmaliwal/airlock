package research

import (
	"testing"

	base "github.com/vmaliwal/airlock/internal/contract"
)

func TestExecuteCompiledContractFailsForUnsupportedBackend(t *testing.T) {
	var c base.Contract
	c.Backend.Kind = "bogus"
	if _, err := ExecuteCompiledContract(c); err == nil {
		t.Fatal("expected unsupported backend error")
	}
}
