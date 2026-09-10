package vroomy

// Flag represents a legacy TOML flag declaration. The current runtime does not
// install or apply these declarations.
type Flag struct {
	Name         string `toml:"name"`
	DefaultValue string `toml:"defaultValue"`
	Usage        string `toml:"usage"`
}
