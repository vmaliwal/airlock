package runner

import (
	"fmt"

	"github.com/vmaliwal/airlock/internal/backend"
	firecrackerbackend "github.com/vmaliwal/airlock/internal/backend/firecracker"
	limabackend "github.com/vmaliwal/airlock/internal/backend/lima"
	"github.com/vmaliwal/airlock/internal/contract"
)

func NewBackend(kind contract.BackendKind) (backend.Backend, error) {
	switch kind {
	case contract.BackendLima:
		return limabackend.Backend{}, nil
	case contract.BackendFirecracker:
		return firecrackerbackend.Backend{}, nil
	default:
		return nil, fmt.Errorf("unsupported backend: %s", kind)
	}
}
