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
