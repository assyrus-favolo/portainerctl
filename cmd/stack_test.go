package cmd

import (
	"strings"
	"testing"
)

func TestGitAuthCredentials(t *testing.T) {
	tests := []struct {
		name            string
		opts            gitAuthOptions
		environment     map[string]string
		stdin           string
		wantUsername    string
		wantPassword    string
		wantAuthType    int
		wantPasswordSet bool
		wantError       bool
	}{
		{
			name:            "credentials from environment",
			environment:     map[string]string{"PORTAINERCTL_GIT_USERNAME": "octocat", "PORTAINERCTL_GIT_PASSWORD": "secret"},
			wantUsername:    "octocat",
			wantPassword:    "secret",
			wantPasswordSet: true,
		},
		{
			name:            "token from stdin",
			opts:            gitAuthOptions{username: "git", authType: "token", passwordStdin: true},
			stdin:           "token-value\n",
			wantUsername:    "git",
			wantPassword:    "token-value",
			wantAuthType:    1,
			wantPasswordSet: true,
		},
		{
			name:        "rejects two password sources",
			opts:        gitAuthOptions{passwordStdin: true},
			environment: map[string]string{"PORTAINERCTL_GIT_PASSWORD": "secret"},
			wantError:   true,
		},
		{
			name:      "rejects unknown auth type",
			opts:      gitAuthOptions{authType: "digest"},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, name := range []string{"PORTAINERCTL_GIT_USERNAME", "PORTAINERCTL_GIT_PASSWORD", "PORTAINERCTL_GIT_AUTH_TYPE"} {
				t.Setenv(name, "")
			}
			for name, value := range test.environment {
				t.Setenv(name, value)
			}

			credential, password, passwordSet, err := test.opts.credentials(strings.NewReader(test.stdin))
			if test.wantError {
				if err == nil {
					t.Fatal("credentials returned no error")
				}
				return
			}
			if err != nil {
				t.Fatalf("credentials returned an error: %v", err)
			}
			if credential.Username != test.wantUsername || password != test.wantPassword || credential.AuthorizationType != test.wantAuthType || passwordSet != test.wantPasswordSet {
				t.Fatalf("credentials returned %#v, password %q, set %t", credential, password, passwordSet)
			}
		})
	}
}
