package firecracker

import (
	"strings"
	"testing"

	"github.com/vmaliwal/airlock/internal/contract"
)

func TestRunRejectsContractsRequiringGuestBinaries(t *testing.T) {
	var c contract.Contract
	c.Backend.Kind = contract.BackendFirecracker
	c.Backend.FirecrackerHost = &contract.FirecrackerHostConfig{Mode: "local"}
	c.Sandbox.NamePrefix = "demo"
	c.Sandbox.ArtifactsDir = "/tmp/demo"
	c.Sandbox.CPU = 2
	c.Sandbox.MemoryGiB = 4
	c.Sandbox.DiskGiB = 10
	c.Repo.CloneURL = "https://github.com/example/repo.git"
	c.Security.Network = contract.NetworkDeny
	c.Security.ExportPaths = []string{"/airlock/artifacts"}
	c.Steps = []contract.Step{{Name: "guest", Run: "/tmp/airlock attempt-run /tmp/x.json"}}

	_, err := Backend{}.Run(c)
	if err == nil || !strings.Contains(err.Error(), "does not yet support guest binary injection") {
		t.Fatalf("expected honest parity error, got %v", err)
	}
}
