package service

const routeFmt = "{ HTTPPath: \"%s\", Target: \"%s\" }"

// Config is the configuration needed to initialize a new instance of Service
type Config struct {
	Port    uint16 `toml:"port"`
	TLSPort uint16 `toml:"tlsPort"`
	TLSDir  string `toml:"tlsDir"`

	// Routes are the routes to listen for and serve
	Routes []*Route `toml:"routes"`
}
