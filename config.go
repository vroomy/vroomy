package vroomy

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/hatchify/errors"
	"github.com/vroomy/httpserve"
)

const (
	// RouteFmt specifies expected route definition syntax
	routeFmt = "{ HTTPPath: \"%s\", Target: \"%s\" Plugin Handler: \"%v\" }"
	// ErrProtectedFlag is returned when a protected flag is used
	ErrProtectedFlag = errors.Error("cannot use protected flag")
)

// NewConfig will return a new configuration
func NewConfig(loc string) (cfg *Config, err error) {
	var c Config
	if _, err = toml.DecodeFile(loc, &c); err != nil {
		return
	}

	if err = c.loadIncludes(); err != nil {
		return
	}

	if c.Dir == "" {
		c.Dir = "./"
	}

	if c.Environment == nil {
		c.Environment = make(map[string]string)
	}

	c.populateFromOSEnv()
	cfg = &c
	return
}

// Config is the configuration needed to initialize a new instance of Service
type Config struct {
	Name string `toml:"name"`

	Dir  string `toml:"dir"`
	Port uint16 `toml:"port"`
	// TLSPort to listen on. To use TLS one of the two must be set:
	//	- TLSDir
	//	- AutoCertHosts/AutoCertDir
	TLSPort uint16 `toml:"tlsPort"`

	TLSDir string `toml:"tlsDir"`

	IncludeConfig

	Flags map[string]string `toml:"-"`

	// Plugin keys as they are referenced by the plugins store
	PluginKeys []string

	ErrorLogger func(error)
}

func (c *Config) hasTLSDir() (ok bool) {
	return len(c.TLSDir) > 0
}

func (c *Config) hasAutoCert() (ok bool) {
	switch {
	case len(c.AutoCertDir) == 0:
		return false
	case len(c.AutoCertHosts) == 0:
		return false

	default:
		return true
	}
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
		var files []fs.DirEntry
		if files, err = os.ReadDir(include); err != nil {
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

// GetGroup will return group with name
func (c *Config) GetRouteGroup(name string) (g *RouteGroup, err error) {
	if len(name) == 0 {
		return
	}

	// TODO: Make this a map for faster lookups?
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

func (c *Config) autoCertConfig() (ac httpserve.AutoCertConfig) {
	ac.DirCache = c.AutoCertDir
	ac.Hosts = c.AutoCertHosts
	return
}

func (c *Config) populateFromOSEnv() {
	for _, kv := range os.Environ() {
		spl := strings.Split(kv, "=")
		if len(spl) < 2 {
			continue
		}

		key := spl[0]
		value := spl[1]
		if _, ok := c.Environment[key]; ok {
			continue
		}

		c.Environment[key] = value
	}
}
