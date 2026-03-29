package research

import (
	base64 "encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	base "github.com/vmaliwal/airlock/internal/contract"
)

func decodeCompiledRunContract(a base.Contract) (RunContract, error) {
	parts := strings.SplitN(a.Steps[0].Run, " ", 2)
	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return RunContract{}, err
	}
	var compiled RunContract
	if err := json.Unmarshal(payload, &compiled); err != nil {
		return RunContract{}, err
	}
	return compiled, nil
}

func TestCompileRunContractSynthesizesPlanFromLocalTargetPath(t *testing.T) {
	repo, err := os.MkdirTemp("", "airlock-compile-local-plan-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(repo)
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rc := RunContract{
		Objective:    "test",
		Mode:         "read_only",
		TargetPath:   repo,
		Airlock:      base.Contract{},
		Reproduction: Phase{Command: "false", Repeat: 1, Success: SuccessRule{MinFailures: pintVal(1)}},
		Validation:   ValidationSpec{TargetCommand: "true", Repeat: 1, Success: SuccessRule{ExitCode: pintVal(0)}},
	}
	rc.Airlock.Backend.Kind = base.BackendLima
	rc.Airlock.Sandbox.NamePrefix = "x"
	rc.Airlock.Sandbox.ArtifactsDir = "/tmp/x"
	rc.Airlock.Sandbox.CPU = 2
	rc.Airlock.Sandbox.MemoryGiB = 2
	rc.Airlock.Sandbox.DiskGiB = 5
	rc.Airlock.Security.Network = base.NetworkDeny

	a, err := CompileRunContract(rc)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := decodeCompiledRunContract(a)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Plan == nil {
		t.Fatalf("expected synthesized plan for local target path, got %#v", compiled)
	}
	if compiled.Plan.TargetRepo != repo {
		t.Fatalf("expected plan target repo %q, got %#v", repo, compiled.Plan)
	}
}
