package vroomy

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vroomy/apex"
)

type testValuePlugin int

func (testValuePlugin) Init(Environment) error { return nil }
func (testValuePlugin) Load(Environment) error { return nil }
func (testValuePlugin) Backend() any           { return nil }
func (testValuePlugin) Close() error           { return nil }

type testEmbeddedFields struct {
	Dependency *string `vroomy:"provider"`
}

type testEmbeddedPlugin struct {
	BasePlugin
	*testEmbeddedFields
}

type testProviderPlugin struct {
	BasePlugin
	value any
}

func (p *testProviderPlugin) Backend() any { return p.value }

type testFactoryPlugin struct{ BasePlugin }

func (*testFactoryPlugin) Empty(...string) (apex.Handler, error) { return nil, nil }

func TestPluginValidation(t *testing.T) {
	type privateFieldPlugin struct {
		BasePlugin
		dependency *string `vroomy:"provider"`
	}

	type embeddedScalar string
	type scalarPlugin struct {
		BasePlugin
		embeddedScalar
	}

	var typedNil *BasePlugin
	var tests = []struct {
		name    string
		plugin  Plugin
		wantErr bool
	}{
		{name: "nil", wantErr: true},
		{name: "typed nil", plugin: typedNil, wantErr: true},
		{name: "value", plugin: testValuePlugin(1), wantErr: true},
		{name: "pointer to scalar", plugin: new(testValuePlugin), wantErr: true},
		{name: "nil embedded pointer", plugin: &testEmbeddedPlugin{}, wantErr: true},
		{name: "private dependency", plugin: &privateFieldPlugin{dependency: new(string)}, wantErr: true},
		{name: "embedded scalar", plugin: &scalarPlugin{embeddedScalar: "value"}},
		{name: "embedded dependency", plugin: &testEmbeddedPlugin{testEmbeddedFields: &testEmbeddedFields{}}},
		{name: "base plugin", plugin: &BasePlugin{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := newPlugins()
			err := registry.Register("test", tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Register() = %v, want error %t", err, tt.wantErr)
			}

			if !tt.wantErr {
				if _, err = makeDependencyMap(tt.plugin); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestDependencyInjection(t *testing.T) {
	value := "backend"
	var tests = []struct {
		name        string
		backend     any
		destination any
		wantErr     bool
	}{
		{name: "exact pointer", backend: &value, destination: new(*string)},
		{name: "interface", backend: &value, destination: new(any)},
		{name: "mismatch", backend: &value, destination: new(int), wantErr: true},
		{name: "nil", destination: new(any), wantErr: true},
		{name: "typed nil", backend: (*string)(nil), destination: new(any), wantErr: true},
		{name: "nil map", backend: map[string]string(nil), destination: new(any), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := Vroomy{pm: map[string]Plugin{"provider": &testProviderPlugin{value: tt.backend}}}
			destination := reflect.ValueOf(tt.destination).Elem()
			err := service.setBackend(destination, "provider")
			if (err != nil) != tt.wantErr {
				t.Fatalf("setBackend() = %v, want error %t", err, tt.wantErr)
			}

			if !tt.wantErr && destination.Interface() != tt.backend {
				t.Fatalf("injected %v, want %v", destination.Interface(), tt.backend)
			}
		})
	}

	consumer := &testEmbeddedPlugin{testEmbeddedFields: &testEmbeddedFields{}}
	service := Vroomy{cfg: &Config{}, pm: map[string]Plugin{
		"provider": &testProviderPlugin{value: &value}, "consumer": consumer,
	}}
	dependencies, err := makeDependencyMap(consumer)
	if err != nil {
		t.Fatal(err)
	}

	if err = service.setDependencies("consumer", dependencies); err != nil {
		t.Fatal(err)
	}

	if consumer.Dependency != &value {
		t.Fatal("embedded dependency was not injected")
	}

	consumer.testEmbeddedFields = nil
	if err = service.setDependencies("consumer", dependencies); err == nil {
		t.Fatal("nil embedded pointer was accepted")
	}
}

func TestDuplicateDependencyFields(t *testing.T) {
	type consumer struct {
		First  *string `vroomy:"provider"`
		Second *string `vroomy:"provider"`
	}

	if _, err := makeDependencyMap(&consumer{}); err == nil || !strings.Contains(err.Error(), "duplicate dependency") {
		t.Fatalf("makeDependencyMap() = %v, want duplicate dependency error", err)
	}
}

func TestPluginErrorContracts(t *testing.T) {
	registry := newPlugins()
	if _, err := registry.Get("missing"); !errors.Is(err, ErrPluginNotLoaded) {
		t.Fatalf("Get() = %v, want missing plugin sentinel", err)
	}

	if err := registry.Register("key", &BasePlugin{}); err != nil {
		t.Fatal(err)
	}

	if err := registry.Register("key", &BasePlugin{}); !errors.Is(err, ErrPluginKeyExists) {
		t.Fatalf("Register() = %v, want duplicate sentinel", err)
	}

	original := p
	p = registry
	t.Cleanup(func() { p = original })
	if err := Register("factory", &testFactoryPlugin{}); err != nil {
		t.Fatal(err)
	}

	if _, err := getHandler("factory.Empty"); !errors.Is(err, ErrInvalidPluginHandler) {
		t.Fatalf("getHandler() = %v, want invalid handler sentinel", err)
	}
}
