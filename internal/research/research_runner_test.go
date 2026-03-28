package research

import (
	"testing"

	base "github.com/vmaliwal/airlock/internal/contract"
)

func TestCompileRunContract(t *testing.T) {
	rc := RunContract{
		Objective:    "test",
		Mode:         "read_only",
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
	rc.Airlock.Repo.CloneURL = "https://github.com/octocat/Hello-World.git"
	rc.Airlock.Security.Network = base.NetworkDeny
	a, err := CompileRunContract(rc)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Steps) != 1 {
		t.Fatalf("expected one compiled step, got %d", len(a.Steps))
	}
}

func pintVal(v int) *int { return &v }
