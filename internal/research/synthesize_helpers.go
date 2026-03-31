package research

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var expectedGotRE = regexp.MustCompile(`expected\s+([A-Za-z0-9_\.\-]+),\s+got\s+([A-Za-z0-9_\.\-]+)`)

func parseExpectedGotPair(s string) (expected, got string, ok bool) {
	m := expectedGotRE.FindStringSubmatch(s)
	if len(m) != 3 {
		return "", "", false
	}
	return m[1], m[2], true
}

func findFileLineContaining(root, needle string) (string, string, bool) {
	var foundPath, foundLine string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".py") && !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".js") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, needle) {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return nil
				}
				foundPath = filepath.ToSlash(rel)
				foundLine = line
				return os.ErrExist
			}
		}
		return nil
	})
	return foundPath, foundLine, foundPath != ""
}

// findFileMultiLineContext finds a file that contains all needles and returns
// a context window around the first needle match, plus the file path.
func findFileMultiLineContext(root string, needles []string, contextLines int) (relPath, context string, found bool) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		for _, needle := range needles {
			if !strings.Contains(content, needle) {
				return nil
			}
		}
		lines := strings.Split(content, "\n")
		// Find line index of first needle
		firstIdx := -1
		for i, line := range lines {
			if strings.Contains(line, needles[0]) {
				firstIdx = i
				break
			}
		}
		if firstIdx < 0 {
			return nil
		}
		start := firstIdx - contextLines
		if start < 0 {
			start = 0
		}
		end := firstIdx + contextLines + 1
		if end > len(lines) {
			end = len(lines)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(rel)
		context = strings.Join(lines[start:end], "\n")
		found = true
		return os.ErrExist
	})
	return relPath, context, found
}

var stopWords = map[string]bool{
	"should": true, "returns": true, "return": true, "function": true,
	"method": true, "field": true, "value": true, "error": true,
	"expected": true, "actual": true, "instead": true, "missing": true,
	"using": true, "issue": true, "cause": true, "because": true,
}

func commonStopWord(w string) bool {
	return stopWords[strings.ToLower(w)]
}
