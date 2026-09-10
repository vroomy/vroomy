package vroomy

// IncludeConfig contains the configuration fields accepted in included TOML files.
type IncludeConfig struct {
	AutoCertHosts []string `toml:"autoCertHosts"`
	AutoCertDir   string   `toml:"autoCertDir"`

	// Application environment
	Environment map[string]string `toml:"env"`

	// Include lists paths to load. Entries merged from included files are appended
	// but are not traversed by the current loadIncludes pass.
	Include []string `toml:"include"`

	// Plugins stores included plugin entries; it does not populate the outer
	// Config.Plugins field or limit which registered plugins run.
	Plugins []string `toml:"plugins"`

	// FlagEntries stores legacy flag declarations; no flag parser consumes them.
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

	if len(merge.AutoCertDir) > 0 {
		i.AutoCertDir = merge.AutoCertDir
		i.AutoCertHosts = merge.AutoCertHosts
	}
}
