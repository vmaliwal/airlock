package contract

import "testing"

func TestValidateContract(t *testing.T) {
	var c Contract
	c.Backend.Kind = BackendLima
	c.Sandbox.NamePrefix = "demo"
	c.Sandbox.ArtifactsDir = "/tmp/demo"
	c.Sandbox.CPU = 2
	c.Sandbox.MemoryGiB = 4
	c.Sandbox.DiskGiB = 10
	c.Repo.CloneURL = "https://github.com/elastic/beats.git"
	c.Security.Network = NetworkDeny
	c.Steps = []Step{{Name: "repro", Run: "go test ./..."}}

	if errs := Validate(c); len(errs) != 0 {
		t.Fatalf("expected valid contract, got %v", errs)
	}
}

func TestValidateContractFirecrackerSSH(t *testing.T) {
	var c Contract
	c.Backend.Kind = BackendFirecracker
	c.Sandbox.NamePrefix = "demo"
	c.Sandbox.ArtifactsDir = "/tmp/demo"
	c.Sandbox.CPU = 2
	c.Sandbox.MemoryGiB = 4
	c.Sandbox.DiskGiB = 10
	c.Repo.CloneURL = "https://github.com/elastic/beats.git"
	c.Security.Network = NetworkAllowlist
	c.Security.AllowHosts = []string{"github.com"}
	c.Steps = []Step{{Name: "repro", Run: "go test ./..."}}

	errs := Validate(c)
	if len(errs) == 0 {
		t.Fatal("expected firecracker host validation errors")
	}
}
