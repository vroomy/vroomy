package vroomy

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gdbu/errors"
	"github.com/vroomy/apex"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// ErrProtectedFlag is a legacy sentinel; the current code does not return it.
	ErrProtectedFlag = errors.Error("cannot use protected flag")
	// ErrInvalidHostPolicy is returned when the HostPolicy does not match the intended signature
	ErrInvalidHostPolicy = errors.Error("invalid HostPolicy handler within the autocert plugin")
)

// NewConfig decodes loc and its includes, defaults Dir to "./", and adds process
// environment entries whose keys are absent from the configured environment.
// File paths are relative to the current working directory, not loc's directory.
func NewConfig(loc string) (cfg *Config, err error) {
	var c Config
	if _, err = toml.DecodeFile(loc, &c); err != nil {
		return nil, fmt.Errorf("decode configuration %q: %w", loc, err)
	}

	if err = c.loadIncludes(); err != nil {
		return nil, err
	}

	if c.Dir == "" {
		c.Dir = "./"
	}

	if c.Environment == nil {
		c.Environment = make(map[string]string)
	}

	c.populateFromOSEnv()
	return &c, nil
}

// Config configures a Vroomy service. NewWithConfig retains and mutates this value.
type Config struct {
	Name string `toml:"name"`

	Dir  string `toml:"dir"`
	Port uint16 `toml:"port"`

	// TLSPort to listen on. To use TLS one of the two must be set:
	//	- TLSDir
	//	- AutoCertHosts/AutoCertDir
	TLSPort uint16 `toml:"tlsPort"`

	TLSDir      string `toml:"tlsDir"`
	AllowNonTLS bool   `toml:"allowNonTLS"`

	IncludeConfig

	// Flags is a legacy field with no active runtime consumer.
	Flags map[string]string `toml:"-"`

	// Plugins stores import-path metadata with no active runtime consumer.
	// It is separate from IncludeConfig.Plugins and does not filter the registry.
	Plugins []string `toml:"plugins"`

	// ErrorLogger receives apex request errors; initialization/listen errors
	// are returned to the caller instead.
	ErrorLogger func(error) `toml:"-"`
}

// GetFilepath returns config.toml within the CONFIG_PATH process environment
// directory, or the current directory if CONFIG_PATH is unset. It does not use c.
func (c *Config) GetFilepath() (filepath string) {
	dir := "."
	configPathEnv, configPathEnvPresent := os.LookupEnv("CONFIG_PATH")
	if configPathEnvPresent {
		dir = configPathEnv
	}

	return path.Join(dir, "config.toml")
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
			return err
		}
	}

	return nil
}

func (c *Config) loadInclude(include string) (err error) {
	if path.Ext(include) == ".toml" {
		// Attempt to decode toml
		var icfg IncludeConfig
		if _, err = toml.DecodeFile(include, &icfg); err != nil {
			return fmt.Errorf("decode include %q: %w", include, err)
		}

		c.IncludeConfig.merge(&icfg)
	} else {
		// Attempt to parse directory
		var files []fs.DirEntry
		if files, err = os.ReadDir(include); err != nil {
			return fmt.Errorf("read include directory %q: %w", include, err)
		}

		// Call recursively
		for _, file := range files {
			if err = c.loadInclude(path.Join(include, file.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetRouteGroup returns the first group matching name, or ErrGroupNotFound.
// An empty name returns nil, nil to select the root group.
func (c *Config) GetRouteGroup(name string) (g *RouteGroup, err error) {
	if len(name) == 0 {
		return g, nil
	}

	// TODO: Make this a map for faster lookups?
	for _, group := range c.Groups {
		if group == nil || group.Name != name {
			continue
		}

		g = group
		return g, nil
	}

	err = ErrGroupNotFound
	return g, err
}

func (c *Config) autoCertConfig() (ac apex.AutoCertConfig, err error) {
	ac.DirCache = c.AutoCertDir
	ac.Hosts = c.AutoCertHosts
	ac.HostPolicy, err = c.getHostPolicy()
	return ac, err
}

func (c *Config) getHostPolicy() (hp autocert.HostPolicy, err error) {
	var primary autocert.HostPolicy
	if primary, err = getHostPolicy(); err != nil {
		return hp, err
	}

	backup := autocert.HostWhitelist(c.AutoCertHosts...)
	if primary == nil {
		primary = backup
	}

	hp = func(ctx context.Context, host string) (policyErr error) {
		if policyErr = primary(ctx, host); policyErr == nil {
			return nil
		}

		if backupErr := backup(ctx, host); backupErr == nil {
			fmt.Printf("Config.getHostPolicy(): failing HostPolicy lookup of <%s>, matched with backup whitelist\n", host)
			return nil
		}

		return policyErr
	}

	return hp, nil
}

func (c *Config) populateFromOSEnv() {
	for _, kv := range os.Environ() {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}

		if _, exists := c.Environment[key]; exists {
			continue
		}

		c.Environment[key] = value
	}
}
