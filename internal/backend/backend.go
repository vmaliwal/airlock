package backend

import "github.com/vmaliwal/airlock/internal/contract"

type RunResult struct {
	SummaryPath string
}

type Backend interface {
	Kind() contract.BackendKind
	CheckPrereqs() []string
	Run(contract.Contract) (RunResult, error)
}
