package vroomy

type Plugin interface {
	Init(env Environment) error
	Load(env Environment) error
	Backend() interface{}
	Close() error
}
