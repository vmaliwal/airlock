package research

import (
	base64 "encoding/base64"
	"encoding/json"
	"strings"
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
		Safety:       SafetyBudget{AllowedPaths: []string{"foo/*"}},
	}
	rc.Airlock.Backend.Kind = base.BackendLima
	rc.Airlock.Sandbox.NamePrefix = "x"
	rc.Airlock.Sandbox.ArtifactsDir = "/tmp/x"
	rc.Airlock.Sandbox.CPU = 2
	rc.Airlock.Sandbox.MemoryGiB = 2
	rc.Airlock.Sandbox.DiskGiB = 5
	rc.Airlock.Repo.CloneURL = "https://github.com/octocat/Hello-World.git"
	rc.Airlock.Repo.Subdir = "pkg/core"
	rc.Airlock.Security.Network = base.NetworkDeny
	a, err := CompileRunContract(rc)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Steps) != 1 {
		t.Fatalf("expected one compiled step, got %d", len(a.Steps))
	}
	if rc.Safety.AllowedPaths[0] != "foo/*" {
		t.Fatalf("input contract should remain unchanged, got %#v", rc.Safety.AllowedPaths)
	}
	parts := strings.SplitN(a.Steps[0].Run, " ", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected compiled step: %q", a.Steps[0].Run)
	}
	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var compiled RunContract
	if err := json.Unmarshal(payload, &compiled); err != nil {
		t.Fatal(err)
	}
	if compiled.Safety.AllowedPaths[0] != "pkg/core/foo/*" {
		t.Fatalf("expected prefixed allowed path, got %#v", compiled.Safety.AllowedPaths)
	}
}

func pintVal(v int) *int { return &v }
