package vroomy

import "fmt"

type Environment map[string]string

func (e Environment) Get(key string) (out string) {
	return e[key]
}

func (e Environment) Must(key string) (out string, err error) {
	var ok bool
	if out, ok = e[key]; !ok {
		err = fmt.Errorf("invalid environment value for <%s>, cannot be empty", key)
		return
	}

	return
}
