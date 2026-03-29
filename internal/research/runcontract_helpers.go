package research

import "os"

func (c RunContract) LocalPlanningTargetPath() string {
	if c.TargetPath == "" {
		return ""
	}
	if _, err := os.Stat(c.TargetPath); err != nil {
		return ""
	}
	return c.TargetPath
}
