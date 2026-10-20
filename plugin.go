package vroomy

// Plugin provides lifecycle hooks and an optional shared backend. Register a
// non-nil pointer to a struct before constructing the service.
type Plugin interface {
	// Init runs before all dependency injection; providers initialize before consumers.
	Init(env Environment) error
	// Load runs after dependencies have been assigned and their plugins loaded.
	Load(env Environment) error
	// Backend returns the value injected into consumers' tagged fields.
	Backend() any
	// Close releases resources after partial or successful Init, in reverse dependency order.
	Close() error
}
