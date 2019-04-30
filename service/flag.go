package service

// Flag represents a flag entry
type Flag struct {
	Name         string `toml:"name"`
	DefaultValue string `toml:"defaultValue"`
	Usage        string `toml:"usage"`
}
