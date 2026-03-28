package env

import "strings"

var sensitivePrefixes = []string{
	"AWS_", "GCP_", "GOOGLE_", "AZURE_", "SSH_", "GITHUB_", "GITLAB_", "OPENAI_", "ANTHROPIC_", "DEEPGRAM_", "NPM_CONFIG_", "BROWSER_", "MEETCLI_",
}

var blocklist = map[string]struct{}{
	"HOME": {}, "USER": {}, "LOGNAME": {}, "SHELL": {}, "PWD": {}, "OLDPWD": {}, "SSH_AUTH_SOCK": {}, "GIT_ASKPASS": {},
	"AWS_PROFILE": {}, "AWS_SHARED_CREDENTIALS_FILE": {}, "AWS_CONFIG_FILE": {}, "NPM_TOKEN": {}, "NODE_AUTH_TOKEN": {}, "GOOGLE_APPLICATION_CREDENTIALS": {},
}

var safeBase = map[string]struct{}{
	"PATH": {}, "LANG": {}, "LC_ALL": {}, "TERM": {},
}

func BuildGuestEnv(hostEnv map[string]string, allowed []string) map[string]string {
	allowedSet := map[string]struct{}{}
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	result := map[string]string{}
	for key, value := range hostEnv {
		if value == "" {
			continue
		}
		if _, blocked := blocklist[key]; blocked {
			continue
		}
		skip := false
		for _, prefix := range sensitivePrefixes {
			if strings.HasPrefix(key, prefix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if _, ok := safeBase[key]; ok {
			result[key] = value
			continue
		}
		if _, ok := allowedSet[key]; ok {
			result[key] = value
		}
	}
	return result
}
