package firecracker

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vmaliwal/airlock/internal/contract"
	"github.com/vmaliwal/airlock/internal/util"
)

type guestBinarySpec struct {
	HostPath  string
	GuestPath string
	Name      string
}

func requiredGuestBinaries(c contract.Contract, workDir string) []guestBinarySpec {
	out := []guestBinarySpec{}
	if contractNeedsBinary(c, "/tmp/airlock-researchguest") {
		out = append(out, guestBinarySpec{
			HostPath:  filepath.Join(workDir, "airlock-researchguest"),
			GuestPath: "/tmp/airlock-researchguest",
			Name:      "airlock-researchguest",
		})
	}
	if contractNeedsBinary(c, "/tmp/airlock") {
		out = append(out, guestBinarySpec{
			HostPath:  filepath.Join(workDir, "airlock"),
			GuestPath: "/tmp/airlock",
			Name:      "airlock",
		})
	}
	return out
}

func contractNeedsBinary(c contract.Contract, guestPath string) bool {
	for _, step := range c.Steps {
		if strings.Contains(step.Run, guestPath) {
			return true
		}
	}
	return false
}

func buildRequiredGuestBinaries(c contract.Contract, workDir string) ([]guestBinarySpec, error) {
	specs := requiredGuestBinaries(c, workDir)
	for _, spec := range specs {
		if err := buildGuestBinary(spec); err != nil {
			return nil, err
		}
	}
	return specs, nil
}

func buildGuestBinary(spec guestBinarySpec) error {
	pkg := "./cmd/" + spec.Name
	goarch := runtime.GOARCH
	if goarch != "amd64" && goarch != "arm64" {
		goarch = "amd64"
	}
	_, err := util.RunLocal("go", []string{"build", "-o", spec.HostPath, pkg}, util.RunOptions{
		Cwd: repoRoot(),
		Env: append(os.Environ(), "GOOS=linux", "GOARCH="+goarch, "CGO_ENABLED=0", "GOTOOLCHAIN=local"),
	})
	if err != nil {
		return fmt.Errorf("build %s guest binary: %w", spec.Name, err)
	}
	return nil
}

func repoRoot() string {
	wd, _ := os.Getwd()
	return wd
}

func copyInArgs(specs []guestBinarySpec) []string {
	args := []string{}
	for _, spec := range specs {
		args = append(args, "--copy-in", spec.HostPath+":"+spec.GuestPath)
	}
	return args
}
