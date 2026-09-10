package vroomy

var _ Plugin = &BasePlugin{}

// BasePlugin supplies no-op lifecycle methods and a nil backend for embedding.
type BasePlugin struct{}

// Init performs no initialization.
func (b *BasePlugin) Init(env Environment) error {
	return nil
}

// Load performs no dependency-dependent setup.
func (b *BasePlugin) Load(env Environment) error {
	return nil
}

// Backend returns nil. Override it when another plugin needs this plugin's backend.
func (b *BasePlugin) Backend() interface{} {
	return nil
}

// Close performs no cleanup.
func (b *BasePlugin) Close() error {
	return nil
}
