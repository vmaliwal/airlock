package main

import (
	base64 "encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vmaliwal/airlock/internal/research"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: researchguest <contract-b64>")
		os.Exit(1)
	}
	payload, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var c research.RunContract
	if err := json.Unmarshal(payload, &c); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	artifacts := os.Getenv("AIRLOCK_ARTIFACTS")
	if artifacts == "" {
		artifacts = "/airlock/artifacts"
	}
	repo := filepath.Join(".")
	if err := research.ExecuteRunContract(c, repo, artifacts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
