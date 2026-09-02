package client

import (
	"net/http"
	"testing"
)

func TestProxyFromEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		environment map[string]string
		wantProxy   string
	}{
		{
			name:        "HTTP proxy",
			target:      "http://portainer.example.com",
			environment: map[string]string{"HTTP_PROXY": "http://proxy.example.com:8080"},
			wantProxy:   "http://proxy.example.com:8080",
		},
		{
			name:        "HTTPS proxy",
			target:      "https://portainer.example.com",
			environment: map[string]string{"HTTPS_PROXY": "http://proxy.example.com:8080"},
			wantProxy:   "http://proxy.example.com:8080",
		},
		{
			name:        "ALL proxy supports socks5h",
			target:      "https://portainer.internal",
			environment: map[string]string{"ALL_PROXY": "socks5h://127.0.0.1:1081"},
			wantProxy:   "socks5h://127.0.0.1:1081",
		},
		{
			name:   "protocol proxy takes precedence over ALL proxy",
			target: "https://portainer.example.com",
			environment: map[string]string{
				"HTTPS_PROXY": "http://https-proxy.example.com:8080",
				"ALL_PROXY":   "socks5h://127.0.0.1:1081",
			},
			wantProxy: "http://https-proxy.example.com:8080",
		},
		{
			name:   "NO proxy bypasses ALL proxy",
			target: "https://portainer.internal",
			environment: map[string]string{
				"ALL_PROXY": "socks5h://127.0.0.1:1081",
				"NO_PROXY":  ".internal",
			},
		},
		{
			name:        "lowercase environment variables",
			target:      "https://portainer.example.com",
			environment: map[string]string{"all_proxy": "socks5://127.0.0.1:1080"},
			wantProxy:   "socks5://127.0.0.1:1080",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearProxyEnvironment(t)
			for name, value := range test.environment {
				t.Setenv(name, value)
			}

			req, err := http.NewRequest(http.MethodGet, test.target, nil)
			if err != nil {
				t.Fatal(err)
			}
			proxyURL, err := proxyFromEnvironment()(req)
			if err != nil {
				t.Fatalf("proxyFromEnvironment returned an error: %v", err)
			}
			if proxyURL == nil {
				if test.wantProxy != "" {
					t.Fatalf("proxyFromEnvironment returned no proxy; want %q", test.wantProxy)
				}
				return
			}
			if got := proxyURL.String(); got != test.wantProxy {
				t.Fatalf("proxyFromEnvironment returned %q; want %q", got, test.wantProxy)
			}
		})
	}
}

func clearProxyEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"HTTP_PROXY", "http_proxy",
		"HTTPS_PROXY", "https_proxy",
		"ALL_PROXY", "all_proxy",
		"NO_PROXY", "no_proxy",
		"REQUEST_METHOD",
	} {
		t.Setenv(name, "")
	}
}
