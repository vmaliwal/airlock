package firecracker

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vmaliwal/airlock/internal/contract"
)

func TestRequiredGuestBinaries(t *testing.T) {
	var c contract.Contract
	c.Backend.Kind = contract.BackendFirecracker
	c.Steps = []contract.Step{{Name: "guest", Run: "/tmp/airlock attempt-run /tmp/x.json && /tmp/airlock-researchguest payload"}}
	specs := requiredGuestBinaries(c, "/tmp/work")
	if len(specs) != 2 {
		t.Fatalf("expected 2 guest binaries, got %#v", specs)
	}
	if specs[0].GuestPath != "/tmp/airlock-researchguest" || specs[1].GuestPath != "/tmp/airlock" {
		t.Fatalf("unexpected guest paths: %#v", specs)
	}
}

func TestCopyInArgs(t *testing.T) {
	specs := []guestBinarySpec{{HostPath: "/tmp/work/airlock", GuestPath: "/tmp/airlock", Name: "airlock"}}
	args := copyInArgs(specs)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--copy-in") || !strings.Contains(joined, "/tmp/work/airlock:/tmp/airlock") {
		t.Fatalf("unexpected copy-in args: %#v", args)
	}
}

func TestFirecrackerRemoteRunCommandIncludesCopyIn(t *testing.T) {
	specs := []guestBinarySpec{{HostPath: filepath.Join("/remote/work", "airlock"), GuestPath: "/tmp/airlock", Name: "airlock"}}
	cmd := firecrackerRemoteRunCommand("demo", "/remote/work", specs)
	if !strings.Contains(cmd, "--copy-in") || !strings.Contains(cmd, "/remote/work/airlock:/tmp/airlock") {
		t.Fatalf("unexpected remote command: %s", cmd)
	}
}
