package vroomy

// Plugin provides lifecycle hooks and an optional shared backend. Register a
// non-nil pointer to a struct before constructing the service.
type Plugin interface {
	// Init runs before dependency injection, in unspecified plugin order.
	Init(env Environment) error
	// Load runs after dependencies have been assigned and their plugins loaded.
	Load(env Environment) error
	// Backend returns the value injected into consumers' tagged fields.
	Backend() interface{}
	// Close releases plugin resources. Plugin close order is unspecified.
	Close() error
}
