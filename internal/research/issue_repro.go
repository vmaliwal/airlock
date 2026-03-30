package research

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var fencedCodeBlockRE = regexp.MustCompile("(?s)```(?:[A-Za-z0-9_+-]+)?\n(.*?)```")

func issueReproSetupSteps(input PlanInput) []SetupStep {
	path, contents, ok := extractIssueReproFile(input.Notes)
	if !ok {
		return nil
	}
	return []SetupStep{{
		Name:    "materialize issue repro file",
		Command: renderWriteFileCommand(path, contents),
	}}
}

func extractIssueReproFile(notes string) (string, string, bool) {
	matches := fencedCodeBlockRE.FindAllStringSubmatch(notes, -1)
	for _, match := range matches {
		block := strings.ReplaceAll(match[1], "\r\n", "\n")
		lines := strings.Split(block, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
			lines = lines[1:]
		}
		if len(lines) == 0 {
			continue
		}
		first := strings.TrimSpace(lines[0])
		path, ok := parseReproPath(first)
		if !ok {
			continue
		}
		body := strings.Join(lines[1:], "\n")
		body = strings.TrimLeft(body, "\n")
		if strings.TrimSpace(body) == "" {
			continue
		}
		return filepath.ToSlash(path), body, true
	}
	return "", "", false
}

func parseReproPath(line string) (string, bool) {
	prefixes := []string{"// ", "# ", "-- ", "; "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			candidate := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if isLikelyRepoRelativeFile(candidate) {
				return candidate, true
			}
		}
	}
	return "", false
}

func isLikelyRepoRelativeFile(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
		return false
	}
	if !strings.Contains(path, "/") || !strings.Contains(filepath.Base(path), ".") {
		return false
	}
	return true
}

func renderWriteFileCommand(path, contents string) string {
	return fmt.Sprintf("mkdir -p %s && cat <<'EOF' > %s\n%s\nEOF", shellQuote(filepath.ToSlash(filepath.Dir(path))), shellQuote(filepath.ToSlash(path)), contents)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
