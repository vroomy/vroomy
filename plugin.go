package vroomy

type Plugin interface {
	Init(env map[string]string) error
	Load(env map[string]string) error
	Backend() interface{}
	Close() error
}
