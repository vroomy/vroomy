package service

import (
	"github.com/BurntSushi/toml"
)

const routeFmt = "{ HTTPPath: \"%s\", Target: \"%s\" Plugin Handler: \"%v\" }"

func newConfig(loc string) (cfg *Config, err error) {
	var c Config
	if _, err = toml.DecodeFile(loc, &c); err != nil {
		return
	}

	if err = c.loadIncludes(); err != nil {
		return
	}

	cfg = &c
	return
}

// Config is the configuration needed to initialize a new instance of Service
type Config struct {
	Dir     string `toml:"dir"`
	Port    uint16 `toml:"port"`
	TLSPort uint16 `toml:"tlsPort"`
	TLSDir  string `toml:"tlsDir"`

	Plugins []string `toml:"plugins"`
	Include []string `toml:"include"`
	IncludeConfig

	// Plugin keys as they are referenced by the plugins store
	pluginKeys []string
}

func (c *Config) loadIncludes() (err error) {
	for _, include := range c.Include {
		var icfg IncludeConfig
		if _, err = toml.DecodeFile(include, &icfg); err != nil {
			return
		}

		c.IncludeConfig.merge(&icfg)
	}

	return
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

// IncludeConfig will include routes
type IncludeConfig struct {
	Groups []*Group `toml:"group"`
	// Routes are the routes to listen for and serve
	Routes []*Route `toml:"route"`
}

func (i *IncludeConfig) merge(merge *IncludeConfig) {
	i.Groups = append(i.Groups, merge.Groups...)
	i.Routes = append(i.Routes, merge.Routes...)
}
