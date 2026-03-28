package runner

import (
	"testing"

	"github.com/vmaliwal/airlock/internal/contract"
)

func TestNewBackend(t *testing.T) {
	if _, err := NewBackend(contract.BackendLima); err != nil {
		t.Fatalf("expected lima backend: %v", err)
	}
	if _, err := NewBackend(contract.BackendFirecracker); err != nil {
		t.Fatalf("expected firecracker backend: %v", err)
	}
	if _, err := NewBackend("bogus"); err == nil {
		t.Fatal("expected unsupported backend error")
	}
}
