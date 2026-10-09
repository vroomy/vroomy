package vroomy

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	gdberrors "github.com/gdbu/errors"
)

type lifecyclePlugin struct {
	BasePlugin
	Dependency *lifecyclePlugin `vroomy:"provider"`
	events     *[]string
	key        string
	initErr    error
	loadErr    error
	closeErr   error
}

func (p *lifecyclePlugin) Init(Environment) error {
	*p.events = append(*p.events, "init "+p.key)
	return p.initErr
}

func (p *lifecyclePlugin) Load(Environment) error {
	*p.events = append(*p.events, "load "+p.key)
	return p.loadErr
}

func (p *lifecyclePlugin) Close() error {
	*p.events = append(*p.events, "close "+p.key)
	return p.closeErr
}

func (p *lifecyclePlugin) Backend() interface{} { return p }

// lifecycleProvider avoids the consumer's dependency tag.
type lifecycleProvider struct {
	BasePlugin
	plugin *lifecyclePlugin
}

func (p *lifecycleProvider) Init(env Environment) error { return p.plugin.Init(env) }
func (p *lifecycleProvider) Load(env Environment) error { return p.plugin.Load(env) }
func (p *lifecycleProvider) Close() error               { return p.plugin.Close() }
func (p *lifecycleProvider) Backend() interface{}       { return p.plugin }

func isolateRegistry(t *testing.T) {
	t.Helper()
	original := p
	p = newPlugins()
	t.Cleanup(func() { p = original })
}

func preserveWorkingDirectory(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if restoreErr := os.Chdir(dir); restoreErr != nil {
			t.Error(restoreErr)
		}
	})
	return dir
}

func TestConstructorRollback(t *testing.T) {
	cause := errors.New("startup failure")
	cleanupCause := errors.New("cleanup failure")
	var tests = []struct {
		name    string
		initErr error
		loadErr error
		routes  []*Route
	}{
		{name: "partial init", initErr: cause},
		{name: "load", loadErr: cause},
		{name: "route", routes: []*Route{{Handlers: []string{"missing.Handler"}, HTTPPath: "/"}}},
		{name: "empty route", routes: []*Route{{}}},
		{name: "nil route", routes: []*Route{nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateRegistry(t)
			previousDir := preserveWorkingDirectory(t)
			var events []string
			provider := &lifecyclePlugin{events: &events, key: "provider"}
			consumer := &lifecyclePlugin{events: &events, key: "consumer", initErr: tt.initErr, loadErr: tt.loadErr, closeErr: cleanupCause}
			if err := Register("provider", &lifecycleProvider{plugin: provider}); err != nil {
				t.Fatal(err)
			}

			if err := Register("consumer", consumer); err != nil {
				t.Fatal(err)
			}

			cfg := &Config{Dir: t.TempDir(), IncludeConfig: IncludeConfig{Routes: tt.routes}}
			service, err := NewWithConfig(cfg)
			if service != nil || err == nil {
				t.Fatalf("NewWithConfig() = %v, %v, want nil and failure", service, err)
			}

			if !errors.Is(err, cleanupCause) {
				t.Fatalf("cleanup cause lost: %v", err)
			}

			if (tt.initErr != nil || tt.loadErr != nil) && !errors.Is(err, cause) {
				t.Fatalf("startup cause lost: %v", err)
			}

			if !slices.Equal(events[len(events)-2:], []string{"close consumer", "close provider"}) {
				t.Fatalf("close order = %v", events)
			}

			if currentDir, dirErr := os.Getwd(); dirErr != nil || currentDir != previousDir {
				t.Fatalf("working directory = %q, %v, want %q", currentDir, dirErr, previousDir)
			}

			consumer.initErr, consumer.loadErr, consumer.closeErr = nil, nil, nil
			cfg.Routes = nil
			service, err = NewWithConfig(cfg)
			if err != nil {
				t.Fatalf("registry not released after failure: %v", err)
			}

			if err = service.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestServicePluginOwnership(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	var events []string
	provider := &lifecyclePlugin{events: &events, key: "provider"}
	consumer := &lifecyclePlugin{events: &events, key: "consumer"}
	if err := Register("provider", &lifecycleProvider{plugin: provider}); err != nil {
		t.Fatal(err)
	}

	if err := Register("consumer", consumer); err != nil {
		t.Fatal(err)
	}

	service, err := NewWithConfig(&Config{Dir: "."})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if !service.closed.Get() {
			if closeErr := service.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}
	})
	if consumer.Dependency != provider {
		t.Fatal("dependency not injected before Load")
	}

	before := slices.Clone(events)
	if _, err = NewWithConfig(&Config{Dir: t.TempDir()}); !errors.Is(err, ErrPluginInUse) {
		t.Fatalf("second service = %v, want exclusive ownership failure", err)
	}

	if err = p.Close(); !errors.Is(err, ErrPluginInUse) {
		t.Fatalf("registry Close() = %v, want ownership failure", err)
	}

	if err = Register("late", &BasePlugin{}); !errors.Is(err, ErrPluginInUse) {
		t.Fatalf("late registration = %v, want ownership failure", err)
	}

	if !slices.Equal(events, before) {
		t.Fatalf("second constructor touched active plugins: %v", events)
	}

	if err = service.Close(); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(events[len(events)-2:], []string{"close consumer", "close provider"}) {
		t.Fatalf("close order = %v", events)
	}

	if err = service.Close(); !errors.Is(err, gdberrors.ErrIsClosed) {
		t.Fatalf("second Close() = %v", err)
	}
}

func TestConstructorValidationBeforeInit(t *testing.T) {
	isolateRegistry(t)
	previousDir := preserveWorkingDirectory(t)
	var events []string
	plugin := &lifecyclePlugin{events: &events, key: "missing provider"}
	if err := Register("consumer", plugin); err != nil {
		t.Fatal(err)
	}

	if _, err := NewWithConfig(&Config{Dir: t.TempDir()}); err == nil {
		t.Fatal("missing dependency accepted")
	}

	if len(events) != 0 {
		t.Fatalf("invalid graph acquired resources: %v", events)
	}

	if dir, err := os.Getwd(); err != nil || dir != previousDir {
		t.Fatalf("invalid graph changed directory: %q, %v", dir, err)
	}

	if _, err := NewWithConfig(nil); err == nil {
		t.Fatal("nil config accepted")
	}
}

func TestFilesystemErrorCauses(t *testing.T) {
	isolateRegistry(t)
	preserveWorkingDirectory(t)
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	if _, err := NewConfig(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewConfig() = %v, want filesystem cause", err)
	}

	if _, err := NewWithConfig(&Config{Dir: missing}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewWithConfig() = %v, want filesystem cause", err)
	}

	cfg := Config{}
	if err := cfg.loadInclude(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("loadInclude() = %v, want filesystem cause", err)
	}
}

func TestDataDirectoryMustBeDirectory(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(filename, nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := initDir(filename); err == nil {
		t.Fatal("existing regular file accepted as data directory")
	}
}
