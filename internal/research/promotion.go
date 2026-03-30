package research

import "fmt"

func PromoteWinningAttempt(repo, attemptName string) (string, bool, error) {
	dirty, err := GitIsDirty(repo)
	if err != nil {
		return "", false, err
	}
	if dirty {
		if err := GitCommitAll(repo, fmt.Sprintf("airlock: promote winning attempt %s", attemptName)); err != nil {
			return "", false, err
		}
		sha, err := GitHeadSHA(repo)
		if err != nil {
			return "", false, err
		}
		return sha, true, nil
	}
	sha, err := GitHeadSHA(repo)
	if err != nil {
		return "", false, err
	}
	return sha, false, nil
}
