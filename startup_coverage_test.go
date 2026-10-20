package vroomy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vroomy/apex"
)

func TestNewFromFile(t *testing.T) {
	var tests = []struct {
		name string
		env  string
		want string
	}{
		{name: "default data directory", want: "data"},
		{name: "explicit data directory", env: "[env]\ndataDir = 'custom'\n", want: "custom"},
		{name: "explicit empty data directory", env: "[env]\ndataDir = ''\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateRegistry(t)
			preserveWorkingDirectory(t)
			t.Setenv("dataDir", "remove-for-default-check")
			if err := os.Unsetenv("dataDir"); err != nil {
				t.Fatal(err)
			}

			var directory = t.TempDir()
			var filename = filepath.Join(directory, "config.toml")
			if err := os.WriteFile(filename, []byte(fmt.Sprintf("dir = %q\n%s", directory, tt.env)), 0600); err != nil {
				t.Fatal(err)
			}

			if err := Register("base", &BasePlugin{}); err != nil {
				t.Fatal(err)
			}

			var service, err = New(filename)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				if closeErr := service.Close(); closeErr != nil {
					t.Error(closeErr)
				}
			})
			if got := service.cfg.Environment["dataDir"]; got != tt.want {
				t.Fatalf("dataDir = %q, want %q", got, tt.want)
			}

			if tt.want != "" {
				if info, statErr := os.Stat(filepath.Join(directory, tt.want)); statErr != nil || !info.IsDir() {
					t.Fatalf("data directory = %v, %v", info, statErr)
				}
			}
		})
	}

	if service, err := New(filepath.Join(t.TempDir(), "missing.toml")); service != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("New(missing) = %v, %v", service, err)
	}
}

func TestConstructorGroupFailures(t *testing.T) {
	var tests = []struct {
		name   string
		groups []*RouteGroup
		routes []*Route
		cause  error
	}{
		{name: "nil group", groups: []*RouteGroup{nil}},
		{name: "missing parent", groups: []*RouteGroup{{Name: "child", Group: "missing"}}, cause: ErrGroupNotFound},
		{name: "parent after child", groups: []*RouteGroup{{Name: "child", Group: "parent"}, {Name: "parent"}}},
		{name: "invalid group handler", groups: []*RouteGroup{{Name: "group", Handlers: []string{"missing.Handler"}}}, cause: ErrPluginNotLoaded},
		{name: "missing route group", routes: []*Route{{Group: "missing", HTTPPath: "/"}}, cause: ErrGroupNotFound},
		{name: "nil route handler", routes: []*Route{{HTTPPath: "/", HTTPHandlers: []apex.Handler{nil}}}, cause: ErrInvalidPluginHandler},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateRegistry(t)
			var previous = preserveWorkingDirectory(t)
			var cfg = &Config{Dir: t.TempDir(), IncludeConfig: IncludeConfig{Groups: tt.groups, Routes: tt.routes}}
			var service, err = NewWithConfig(cfg)
			if service != nil || err == nil {
				t.Fatalf("NewWithConfig() = %v, %v, want failure", service, err)
			}

			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Fatalf("error = %v, want cause %v", err, tt.cause)
			}

			if directory, dirErr := os.Getwd(); dirErr != nil || directory != previous {
				t.Fatalf("working directory = %q, %v, want %q", directory, dirErr, previous)
			}

			if p.inUse {
				t.Fatal("failed constructor retained registry ownership")
			}
		})
	}
}

func TestConstructorDependencyFailures(t *testing.T) {
	type duplicatePlugin struct {
		BasePlugin
		First  any `vroomy:"provider"`
		Second any `vroomy:"provider"`
	}

	var tests = []struct {
		name   string
		plugin Plugin
	}{
		{name: "invalid registered instance"},
		{name: "duplicate dependency", plugin: &duplicatePlugin{}},
		{name: "backend type mismatch", plugin: &testEmbeddedPlugin{testEmbeddedFields: &testEmbeddedFields{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateRegistry(t)
			preserveWorkingDirectory(t)
			// Simulate legacy invalid registry entries to check startup validation.
			p.pm["consumer"] = tt.plugin
			p.pm["provider"] = &testProviderPlugin{value: 1}
			if service, err := NewWithConfig(&Config{Dir: "."}); service != nil || err == nil {
				t.Fatalf("NewWithConfig() = %v, %v, want dependency failure", service, err)
			}
		})
	}
}

func TestConstructorDataDirectoryFailure(t *testing.T) {
	isolateRegistry(t)
	var previous = preserveWorkingDirectory(t)
	var directory = t.TempDir()
	var cfg = &Config{Dir: directory, IncludeConfig: IncludeConfig{Environment: Environment{"dataDir": "missing/child"}}}
	if service, err := NewWithConfig(cfg); service != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewWithConfig() = %v, %v", service, err)
	}

	if current, err := os.Getwd(); err != nil || current != previous || p.inUse {
		t.Fatalf("rollback state: directory %q, %v; registry owned %t", current, err, p.inUse)
	}
}

func TestInitRoutesInitializesDeferredGroup(t *testing.T) {
	var tests = []struct {
		name     string
		handlers []string
		wantErr  bool
	}{
		{name: "valid group"},
		{name: "invalid group", handlers: []string{"missing.Handler"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateRegistry(t)
			var group = &RouteGroup{Name: "deferred", HTTPPath: "/group", Handlers: tt.handlers}
			var service = Vroomy{srv: apex.New(), cfg: &Config{IncludeConfig: IncludeConfig{
				Groups: []*RouteGroup{group}, Routes: []*Route{{Group: "deferred", HTTPPath: "/route"}},
			}}}
			var err = service.initRoutes()
			if (err != nil) != tt.wantErr {
				t.Fatalf("initRoutes() = %v, want error %t", err, tt.wantErr)
			}

			if !tt.wantErr && group.G == nil {
				t.Fatal("deferred group was not initialized")
			}
		})
	}
}

func TestGetPluginMethodRejectsInvalidRegistryEntry(t *testing.T) {
	isolateRegistry(t)
	p.pm["invalid"] = nil
	if _, err := getPluginMethod("invalid", "Load"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("getPluginMethod() = %v, want invalid plugin context", err)
	}
}

func TestBasePluginBackend(t *testing.T) {
	var plugin BasePlugin
	if backend := plugin.Backend(); backend != nil {
		t.Fatalf("Backend() = %v, want nil", backend)
	}
}

func TestConfiguredPorts(t *testing.T) {
	var service = Vroomy{cfg: &Config{Port: 8080, TLSPort: 8443}}
	if service.Port() != 8080 || service.TLSPort() != 8443 {
		t.Fatalf("ports = %d, %d", service.Port(), service.TLSPort())
	}
}

func TestListenUntilSignalDeadline(t *testing.T) {
	isolateRegistry(t)
	var service, err = NewWithConfig(&Config{Dir: "."})
	if err != nil {
		t.Fatal(err)
	}

	var ctx, cancel = context.WithDeadline(context.Background(), time.Time{})
	defer cancel()
	if err = service.ListenUntilSignal(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ListenUntilSignal() = %v, want deadline", err)
	}
}

func TestConstructorWithRemovedWorkingDirectory(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	var directory = t.TempDir()
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(directory); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Getwd(); err == nil {
		t.Skip("platform can still resolve a removed working directory")
	}

	if service, err := NewWithConfig(&Config{Dir: "."}); service != nil || !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "get working directory") {
		t.Fatalf("NewWithConfig() = %v, %v, want getwd failure", service, err)
	}

	if p.inUse {
		t.Fatal("getwd failure retained registry ownership")
	}
}

type removePreviousDirectoryPlugin struct {
	BasePlugin
	previous string
	cause    error
}

func (p *removePreviousDirectoryPlugin) Init(Environment) error {
	if err := os.Remove(p.previous); err != nil {
		return err
	}

	return p.cause
}

func TestConstructorDirectoryRestoreFailure(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	var previous = t.TempDir()
	var target = t.TempDir()
	if err := os.Chdir(previous); err != nil {
		t.Fatal(err)
	}

	var cause = errors.New("init failure")
	if err := Register("remove", &removePreviousDirectoryPlugin{previous: previous, cause: cause}); err != nil {
		t.Fatal(err)
	}

	var service, err = NewWithConfig(&Config{Dir: target})
	if service != nil || !errors.Is(err, cause) || !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "restore working directory") {
		t.Fatalf("NewWithConfig() = %v, %v, want init and restoration failures", service, err)
	}

	if p.inUse {
		t.Fatal("restore failure retained registry ownership")
	}
}
