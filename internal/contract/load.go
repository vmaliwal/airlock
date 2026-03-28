package contract

import (
	"encoding/json"
	"fmt"
	"os"
)

func Load(path string) (Contract, error) {
	var c Contract
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

func Validate(c Contract) []string {
	var errs []string
	if c.Backend.Kind == "" {
		errs = append(errs, "backend.kind is required")
	}
	if c.Sandbox.NamePrefix == "" {
		errs = append(errs, "sandbox.namePrefix is required")
	}
	if c.Sandbox.ArtifactsDir == "" {
		errs = append(errs, "sandbox.artifactsDir is required")
	}
	if c.Sandbox.CPU <= 0 {
		errs = append(errs, "sandbox.cpu must be > 0")
	}
	if c.Sandbox.MemoryGiB <= 0 {
		errs = append(errs, "sandbox.memoryGiB must be > 0")
	}
	if c.Sandbox.DiskGiB <= 0 {
		errs = append(errs, "sandbox.diskGiB must be > 0")
	}
	if c.Repo.CloneURL == "" {
		errs = append(errs, "repo.cloneUrl is required")
	}
	if len(c.Steps) == 0 {
		errs = append(errs, "at least one step is required")
	}
	if c.Security.Network == NetworkAllowlist && len(c.Security.AllowHosts) == 0 {
		errs = append(errs, "security.allowHosts is required when network=allowlist")
	}
	if c.Security.BootstrapNetwork == NetworkAllowlist && len(c.Security.BootstrapAllowHosts) == 0 {
		errs = append(errs, "security.bootstrapAllowHosts is required when bootstrapNetwork=allowlist")
	}
	if c.Backend.Kind == BackendFirecracker {
		if c.Backend.FirecrackerHost == nil {
			errs = append(errs, "backend.firecrackerHost is required for firecracker backend")
		} else if c.Backend.FirecrackerHost.Mode == "ssh" {
			if c.Backend.FirecrackerHost.Host == "" || c.Backend.FirecrackerHost.User == "" || c.Backend.FirecrackerHost.RemoteWorkDir == "" {
				errs = append(errs, "firecracker ssh mode requires host, user, and remoteWorkDir")
			}
		}
	}
	for i, step := range c.Steps {
		if step.Name == "" {
			errs = append(errs, fmt.Sprintf("steps[%d].name is required", i))
		}
		if step.Run == "" {
			errs = append(errs, fmt.Sprintf("steps[%d].run is required", i))
		}
	}
	return errs
}
