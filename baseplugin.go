package vroomy

var _ Plugin = &BasePlugin{}

type BasePlugin struct{}

func (b *BasePlugin) Init(env Environment) error {
	return nil
}

func (b *BasePlugin) Load(env Environment) error {
	return nil
}

func (b *BasePlugin) Backend() interface{} {
	return nil
}

func (b *BasePlugin) Close() error {
	return nil
}
