package service

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/BurntSushi/toml"
)

const routeFmt = "{ HTTPPath: \"%s\", Target: \"%s\" Plugin Handler: \"%v\" }"

// NewConfig will return a new configuration
func NewConfig(loc string) (cfg *Config, err error) {
	var c Config
	if _, err = toml.DecodeFile(loc, &c); err != nil {
		return
	}

	if err = c.loadIncludes(); err != nil {
		return
	}

	if err = c.initFlags(); err != nil {
		return
	}

	if c.Dir == "" {
		c.Dir = "./"
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

	Flags map[string]string `toml:"-"`

	FlagEntries []Flag `toml:"flag"`

	PerformUpdate bool `toml:"-"`

	// Plugin keys as they are referenced by the plugins store
	pluginKeys []string
}

func (c *Config) loadIncludes() (err error) {
	for _, include := range c.Include {
		// Include each file or directory
		if err = c.loadInclude(include); err != nil {
			// Include failed
			return
		}
	}

	return
}

func (c *Config) loadInclude(include string) (err error) {
	if path.Ext(include) == ".toml" {
		// Attempt to decode toml
		var icfg IncludeConfig
		if _, err = toml.DecodeFile(include, &icfg); err != nil {
			return
		}

		c.IncludeConfig.merge(&icfg)
	} else {
		// Attempt to parse directory
		var files []os.FileInfo
		if files, err = ioutil.ReadDir(include); err != nil {
			return fmt.Errorf("%s is not a .toml file or directory", include)
		}

		// Call recursively
		for _, file := range files {
			if err = c.loadInclude(path.Join(include, file.Name())); err != nil {
				return
			}
		}
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

func (c *Config) initFlags() (err error) {
	var strs []*string
	flag.BoolVar(&c.PerformUpdate, "update", false, "Whether or not to update all plugins on start-up")

	for _, flagEntry := range c.FlagEntries {
		switch flagEntry.Name {
		case "update":
			err = fmt.Errorf("error setting flag \"%s\": %v", flagEntry.Name, ErrProtectedFlag)
			return
		}

		str := flag.String(flagEntry.Name, flagEntry.DefaultValue, flagEntry.Usage)
		strs = append(strs, str)
	}

	flag.Parse()
	c.Flags = make(map[string]string, len(strs))

	for i, str := range strs {
		c.Flags[c.FlagEntries[i].Name] = *str
	}

	return
}

// IncludeConfig will include routes
type IncludeConfig struct {
	// Application environment
	Environment map[string]string `toml:"env"`
	// Groups are the route groups
	Groups []*Group `toml:"group"`
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

	i.Groups = append(i.Groups, merge.Groups...)
	i.Routes = append(i.Routes, merge.Routes...)
}
