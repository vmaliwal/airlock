package research

import (
	"strings"
	"testing"
)

func TestAirlockVersionInjectedBuildVersion(t *testing.T) {
	old := BuildVersion
	defer func() { BuildVersion = old }()
	BuildVersion = "v1.2.3"
	if got := AirlockVersion(); got != "v1.2.3" {
		t.Fatalf("expected injected version v1.2.3, got %q", got)
	}
}

func TestAirlockVersionFallbackIsNotEmpty(t *testing.T) {
	old := BuildVersion
	defer func() { BuildVersion = old }()
	BuildVersion = ""
	got := AirlockVersion()
	if strings.TrimSpace(got) == "" {
		t.Fatalf("expected non-empty version fallback, got empty string")
	}
	// When running under 'go test' without link-time injection and without
	// a tagged commit, should be "dev" or "dev-<hash>..." — never "(devel)".
	if strings.Contains(got, "(devel)") {
		t.Fatalf("version should never expose raw go module (devel): %q", got)
	}
}
