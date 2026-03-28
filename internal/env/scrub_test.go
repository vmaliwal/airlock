package env

import "testing"

func TestBuildGuestEnv(t *testing.T) {
	host := map[string]string{
		"PATH":          "/usr/bin",
		"LANG":          "C.UTF-8",
		"HOME":          "/Users/varun",
		"SSH_AUTH_SOCK": "/tmp/agent.sock",
		"AWS_PROFILE":   "prod",
		"FOO":           "bar",
	}
	got := BuildGuestEnv(host, []string{"FOO"})
	if got["PATH"] != "/usr/bin" || got["LANG"] != "C.UTF-8" || got["FOO"] != "bar" {
		t.Fatalf("missing expected values: %#v", got)
	}
	if _, ok := got["HOME"]; ok {
		t.Fatal("HOME should be scrubbed")
	}
	if _, ok := got["SSH_AUTH_SOCK"]; ok {
		t.Fatal("SSH_AUTH_SOCK should be scrubbed")
	}
	if _, ok := got["AWS_PROFILE"]; ok {
		t.Fatal("AWS_PROFILE should be scrubbed")
	}
}
