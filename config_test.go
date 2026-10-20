package vroomy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func TestNewConfig_IncludePathsAndNonRecursion(t *testing.T) {
	preserveWorkingDirectory(t)
	workingDir := t.TempDir()
	configDir := filepath.Join(workingDir, "config")
	includeDir := filepath.Join(workingDir, "includes")
	if err := os.Mkdir(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(includeDir, 0700); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(configDir, "main.toml"): "dir = 'not-applied-by-NewConfig'\ninclude = ['includes']\n[[group]]\nname = 'main'\nhttpPath = '/main'\n",
		filepath.Join(includeDir, "a.toml"):   "include = ['missing-nested.toml']\n[env]\nsource = 'first'\n[[group]]\nname = 'first'\nhttpPath = '/first'\n",
		filepath.Join(includeDir, "b.toml"):   "[env]\nsource = 'second'\n[[group]]\nname = 'second'\nhttpPath = '/second'\n",
	}
	for filename, contents := range files {
		if err := os.WriteFile(filename, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Chdir(workingDir); err != nil {
		t.Fatal(err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := NewConfig(filepath.Join("config", "main.toml"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Environment["source"] != "second" {
		t.Fatalf("directory includes did not follow filename order: %v", cfg.Environment["source"])
	}

	var groupNames []string
	for _, group := range cfg.Groups {
		groupNames = append(groupNames, group.Name)
	}

	if !slices.Equal(groupNames, []string{"main", "first", "second"}) {
		t.Fatalf("group order = %v", groupNames)
	}

	if !slices.Equal(cfg.Include, []string{"includes", "missing-nested.toml"}) {
		t.Fatalf("nested include metadata = %v", cfg.Include)
	}

	if currentDir, dirErr := os.Getwd(); dirErr != nil || currentDir != workingDir {
		t.Fatalf("NewConfig changed working directory: %q, %v", currentDir, dirErr)
	}
}

func TestNewConfigDefaultsAndFailures(t *testing.T) {
	var dir = t.TempDir()
	var empty = filepath.Join(dir, "empty.toml")
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}

	var cfg, err = NewConfig(empty)
	if err != nil || cfg.Dir != "./" || cfg.Environment == nil {
		t.Fatalf("empty config = %v, %v", cfg, err)
	}

	var tests = []struct {
		name     string
		contents string
		cause    error
	}{
		{name: "invalid main TOML", contents: "port = ["},
		{name: "missing included file", contents: fmt.Sprintf("include = [%q]", filepath.Join(dir, "missing.toml")), cause: os.ErrNotExist},
		{name: "invalid included TOML", contents: fmt.Sprintf("include = [%q]", filepath.Join(dir, "invalid.toml"))},
		{name: "directory containing non TOML file", contents: fmt.Sprintf("include = [%q]", filepath.Join(dir, "includes"))},
	}
	if err = os.WriteFile(filepath.Join(dir, "invalid.toml"), []byte("["), 0600); err != nil {
		t.Fatal(err)
	}

	var includes = filepath.Join(dir, "includes")
	if err = os.Mkdir(includes, 0700); err != nil {
		t.Fatal(err)
	}

	if err = os.WriteFile(filepath.Join(includes, "readme.txt"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filename = filepath.Join(dir, "main.toml")
			if writeErr := os.WriteFile(filename, []byte(tt.contents), 0600); writeErr != nil {
				t.Fatal(writeErr)
			}

			var result, loadErr = NewConfig(filename)
			if result != nil || loadErr == nil {
				t.Fatalf("NewConfig() = %v, %v, want failure", result, loadErr)
			}

			if tt.cause != nil && !errors.Is(loadErr, tt.cause) {
				t.Fatalf("error = %v, want cause %v", loadErr, tt.cause)
			}
		})
	}
}

func TestConfigGetFilepath(t *testing.T) {
	var cfg Config
	t.Setenv("CONFIG_PATH", "restore-after-test")
	if err := os.Unsetenv("CONFIG_PATH"); err != nil {
		t.Fatal(err)
	}

	if got := cfg.GetFilepath(); got != "config.toml" {
		t.Fatalf("unset CONFIG_PATH = %q", got)
	}

	for _, directory := range []string{"", "/etc/vroomy", "relative/config"} {
		t.Setenv("CONFIG_PATH", directory)
		if got, want := cfg.GetFilepath(), filepath.Join(directory, "config.toml"); got != want {
			t.Errorf("CONFIG_PATH %q = %q, want %q", directory, got, want)
		}
	}
}

func TestConfigCertificateSources(t *testing.T) {
	var tests = []struct {
		name  string
		dir   string
		hosts []string
		want  bool
	}{
		{name: "neither"},
		{name: "hosts only", hosts: []string{"example.com"}},
		{name: "directory only", dir: "cache"},
		{name: "both", dir: "cache", hosts: []string{"example.com"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg = Config{IncludeConfig: IncludeConfig{AutoCertDir: tt.dir, AutoCertHosts: tt.hosts}}
			if got := cfg.hasAutoCert(); got != tt.want {
				t.Fatalf("hasAutoCert() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestConfigGetRouteGroup(t *testing.T) {
	var first = &RouteGroup{Name: "duplicate"}
	var cfg = Config{IncludeConfig: IncludeConfig{Groups: []*RouteGroup{nil, first, {Name: "duplicate"}}}}
	if group, err := cfg.GetRouteGroup("duplicate"); err != nil || group != first {
		t.Fatalf("GetRouteGroup() = %v, %v, want first match", group, err)
	}

	if group, err := cfg.GetRouteGroup("missing"); group != nil || !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("missing group = %v, %v", group, err)
	}
}
