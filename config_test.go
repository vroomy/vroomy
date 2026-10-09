package vroomy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/acme/autocert"
)

func TestConfig_populateFromOSEnv(t *testing.T) {
	type testcase struct {
		name       string
		process    string
		configured Environment
		want       string
	}

	const key = "VROOMY_TEST_ENV_VALUE"
	var tests = []testcase{
		{name: "multiple equals signs", process: "abc=def=ghi", want: "abc=def=ghi"},
		{name: "leading equals sign", process: "=abc", want: "=abc"},
		{name: "empty process value"},
		{name: "configured value wins", process: "process", configured: Environment{key: "configured=content"}, want: "configured=content"},
		{name: "configured empty value wins", process: "process", configured: Environment{key: ""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(key, tc.process)
			var cfg Config
			cfg.Environment = tc.configured
			if cfg.Environment == nil {
				cfg.Environment = make(Environment)
			}

			cfg.populateFromOSEnv()
			var got, exists = cfg.Environment[key]
			if !exists || got != tc.want {
				t.Errorf("environment value = (%q, %t), want (%q, true)", got, exists, tc.want)
			}
		})
	}
}

func TestConfig_getHostPolicy(t *testing.T) {
	type testcase struct {
		name        string
		provider    bool
		host        string
		wantErr     bool
		wantPrimary bool
	}

	var primaryErr = errors.New("primary rejected host")
	var tests = []testcase{
		{name: "missing provider uses allowlist", host: "backup.example"},
		{name: "missing provider rejects unlisted host", host: "unlisted.example", wantErr: true},
		{name: "primary accepts unlisted host", provider: true, host: "primary.example"},
		{name: "allowlist accepts primary rejection", provider: true, host: "backup.example"},
		{name: "both reject preserves primary error", provider: true, host: "unlisted.example", wantErr: true, wantPrimary: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var original = p
			p = newPlugins()
			t.Cleanup(func() { p = original })
			if tc.provider {
				if registerErr := p.Register("autocert", &hostPolicyTestPlugin{accepted: "primary.example", err: primaryErr}); registerErr != nil {
					t.Fatal(registerErr)
				}
			}

			var cfg = Config{IncludeConfig: IncludeConfig{AutoCertHosts: []string{"backup.example"}}}
			var (
				policy autocert.HostPolicy
				err    error
			)
			if policy, err = cfg.getHostPolicy(); err != nil {
				t.Fatal(err)
			}

			var policyErr = policy(context.Background(), tc.host)
			if (policyErr != nil) != tc.wantErr {
				t.Fatalf("policy(%q) error = %v, want error %t", tc.host, policyErr, tc.wantErr)
			}

			if tc.wantPrimary && !errors.Is(policyErr, primaryErr) {
				t.Errorf("policy(%q) error = %v, want primary error %v", tc.host, policyErr, primaryErr)
			}
		})
	}
}

type hostPolicyTestPlugin struct {
	BasePlugin
	accepted string
	err      error
}

func (p *hostPolicyTestPlugin) HostPolicy(ctx context.Context, host string) error {
	if host == p.accepted {
		return nil
	}

	return p.err
}

func TestNewConfig_EnvironmentPrecedence(t *testing.T) {
	const (
		sharedKey  = "VROOMY_TEST_SHARED"
		mainKey    = "VROOMY_TEST_MAIN"
		firstKey   = "VROOMY_TEST_FIRST"
		processKey = "VROOMY_TEST_PROCESS"
	)
	t.Setenv(sharedKey, "process")
	t.Setenv(mainKey, "process")
	t.Setenv(firstKey, "process")
	t.Setenv(processKey, "abc=def=ghi")

	var (
		dir    = t.TempDir()
		first  = filepath.Join(dir, "first.toml")
		second = filepath.Join(dir, "second.toml")
		main   = filepath.Join(dir, "main.toml")
	)
	var files = map[string]string{
		first:  fmt.Sprintf("[env]\n%s = %q\n%s = %q\n", sharedKey, "first", firstKey, "first"),
		second: fmt.Sprintf("[env]\n%s = %q\n", sharedKey, "second"),
		main:   fmt.Sprintf("include = [%q, %q]\n[env]\n%s = %q\n%s = %q\n", first, second, sharedKey, "main", mainKey, "main"),
	}
	for filename, contents := range files {
		if err := os.WriteFile(filename, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}

	var (
		cfg *Config
		err error
	)
	if cfg, err = NewConfig(main); err != nil {
		t.Fatal(err)
	}

	var want = Environment{sharedKey: "second", mainKey: "main", firstKey: "first", processKey: "abc=def=ghi"}
	for key, value := range want {
		if got := cfg.Environment[key]; got != value {
			t.Errorf("Environment[%q] = %q, want %q", key, got, value)
		}
	}

	if cfg.Dir != "./" {
		t.Errorf("default Dir = %q, want ./", cfg.Dir)
	}
}
