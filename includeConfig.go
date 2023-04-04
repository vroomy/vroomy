package vroomy

// IncludeConfig will include routes
type IncludeConfig struct {
	// Application environment
	Environment map[string]string `toml:"env"`

	// Allow included files to add includes
	Include []string `toml:"include"`

	// Specify which plugins are in scope
	Plugins []string `toml:"plugins"`

	// Flags are the dynamic flags specified in config
	FlagEntries []*Flag `toml:"flag"`

	// Groups are the route groups
	Groups []*RouteGroup `toml:"group"`
	// Routes are the routes to listen for and serve
	Routes []*Route `toml:"route"`
}

func (i *IncludeConfig) merge(merge *IncludeConfig) {
	if i.Environment == nil {
		i.Environment = make(map[string]string)
	}

	for key, val := range merge.Environment {
		i.Environment[key] = val
	}

	i.Include = append(i.Include, merge.Include...)

	i.Plugins = append(i.Plugins, merge.Plugins...)

	i.FlagEntries = append(i.FlagEntries, merge.FlagEntries...)

	i.Groups = append(i.Groups, merge.Groups...)
	i.Routes = append(i.Routes, merge.Routes...)
}
