package research

func (c RunContract) TargetPathOrRepoCloneURLHint() string {
	if c.TargetPath != "" {
		return c.TargetPath
	}
	if c.Airlock.Repo.Subdir != "" {
		return c.Airlock.Repo.Subdir
	}
	return "."
}
