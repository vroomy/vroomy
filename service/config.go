package service

const routeFmt = "{ HTTPPath: \"%s\", Target: \"%s\" Plugin Handler: \"%v\" }"

// Config is the configuration needed to initialize a new instance of Service
type Config struct {
	Dir     string   `toml:"dir"`
	Port    uint16   `toml:"port"`
	TLSPort uint16   `toml:"tlsPort"`
	TLSDir  string   `toml:"tlsDir"`
	Plugins []string `toml:"plugins"`

	Groups []*Group `toml:"group"`

	// Routes are the routes to listen for and serve
	Routes []*Route `toml:"route"`
}

func (c *Config) getGroup(name string) (g *Group, err error) {
	if len(name) == 0 {
		return
	}

	for _, group := range c.Groups {
		if group.Name != name {
			continue
		}

		g = group
		return
	}

	err = ErrGroupNotFound
	return
}
