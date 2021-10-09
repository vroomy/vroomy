package vroomy

type Plugin interface {
	Init(env map[string]string) error
	Load() error
	Backend() interface{}
	Close() error
}
