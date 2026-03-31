package research

import (
	"runtime/debug"
	"strings"
)

// BuildVersion is injected at link time via:
//
//	-ldflags "-X github.com/vmaliwal/airlock/internal/research.BuildVersion=v1.2.3"
var BuildVersion string

func AirlockVersion() string {
	// 1. Injected at link time (release builds via install.sh or go install)
	if v := strings.TrimSpace(BuildVersion); v != "" {
		return v
	}
	// 2. Module version from go build metadata (go install from a tagged module)
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := strings.TrimSpace(info.Main.Version); v != "" && v != "(devel)" {
			return v
		}
		// 3. VCS tag from build settings (go run in a git repo with a tag)
		for _, s := range info.Settings {
			if s.Key == "vcs.tag" && strings.TrimSpace(s.Value) != "" {
				return strings.TrimSpace(s.Value)
			}
		}
		// 4. Short VCS commit hash as fallback identifier
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				dirty := ""
				for _, t := range info.Settings {
					if t.Key == "vcs.modified" && t.Value == "true" {
						dirty = "-dirty"
					}
				}
				return "dev-" + s.Value[:7] + dirty
			}
		}
	}
	return "dev"
}
