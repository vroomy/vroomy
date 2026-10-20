package vroomy

import (
	"errors"
	"testing"

	gdberrors "github.com/gdbu/errors"
)

func TestPluginsRegistryLifecycle(t *testing.T) {
	var events []string
	var firstCause = errors.New("first close failure")
	var secondCause = errors.New("second close failure")
	var registry = newPlugins()
	var plugins = map[string]Plugin{
		"first":  &lifecycleProvider{plugin: &lifecyclePlugin{events: &events, key: "first", closeErr: firstCause}},
		"second": &lifecycleProvider{plugin: &lifecyclePlugin{events: &events, key: "second", closeErr: secondCause}},
		"base":   &BasePlugin{},
	}
	for key, plugin := range plugins {
		if err := registry.Register(key, plugin); err != nil {
			t.Fatal(err)
		}

		if got, err := registry.Get(key); err != nil || got != plugin {
			t.Fatalf("Get(%q) = %v, %v, want registered instance", key, got, err)
		}
	}

	var loaded = registry.Loaded()
	delete(loaded, "first")
	loaded["extra"] = &BasePlugin{}
	if got := registry.Loaded(); len(got) != 3 || got["first"] != plugins["first"] || got["extra"] != nil {
		t.Fatalf("Loaded exposed registry map: %v", got)
	}

	if err := registry.Test(); err == nil {
		t.Fatal("Test should report that it is unimplemented")
	}

	if err := registry.TestAsync(nil); err == nil {
		t.Fatal("TestAsync should report that it is unimplemented")
	}

	var closeErr = registry.Close()
	if !errors.Is(closeErr, firstCause) || !errors.Is(closeErr, secondCause) || len(events) != 2 {
		t.Fatalf("Close() = %v, events = %v; want both cleanup causes and calls", closeErr, events)
	}

	if err := registry.Close(); !errors.Is(err, gdberrors.ErrIsClosed) {
		t.Fatalf("repeated Close() = %v", err)
	}

	if err := registry.Register("late", &BasePlugin{}); !errors.Is(err, gdberrors.ErrIsClosed) {
		t.Fatalf("closed Register() = %v", err)
	}

	if plugin, err := registry.Get("base"); plugin != nil || !errors.Is(err, gdberrors.ErrIsClosed) {
		t.Fatalf("closed Get() = %v, %v", plugin, err)
	}

	if snapshot, err := registry.acquire(); snapshot != nil || !errors.Is(err, gdberrors.ErrIsClosed) {
		t.Fatalf("closed acquire() = %v, %v", snapshot, err)
	}
}

func TestPluginsInvalidKeys(t *testing.T) {
	for _, key := range []string{"", "with.dot", "with(paren", "with)paren"} {
		t.Run(key, func(t *testing.T) {
			var registry = newPlugins()
			if err := registry.Register(key, &BasePlugin{}); err == nil {
				t.Fatalf("Register(%q) accepted invalid key", key)
			}

			if len(registry.Loaded()) != 0 {
				t.Fatal("failed registration changed registry")
			}
		})
	}
}
