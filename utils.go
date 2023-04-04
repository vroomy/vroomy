package vroomy

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/vroomy/httpserve"
)

type listener interface {
	Listen(port uint16) error
}

func initDir(loc string) (err error) {
	if len(loc) == 0 {
		return
	}

	if err = os.Mkdir(loc, 0744); err == nil {
		return
	}

	if os.IsExist(err) {
		return nil
	}

	return
}

func getHandlerParts(handlerKey string) (key, handler string, args []string, err error) {
	spl := strings.SplitN(handlerKey, ".", 2)
	if len(spl) != 2 {
		err = fmt.Errorf("expected key and handler, received \"%s\"", handlerKey)
		return
	}

	key = spl[0]
	handler = spl[1]

	spl = strings.Split(handler, "(")
	if len(spl) == 1 {
		return
	}

	handler = spl[0]
	argsStr := spl[1]

	if argsStr[len(argsStr)-1] != ')' {
		err = ErrExpectedEndParen
		return
	}

	argsStr = argsStr[:len(argsStr)-1]
	args = strings.Split(argsStr, ",")
	return
}

func getHandler(handlerKey string) (h httpserve.Handler, err error) {
	var (
		key     string
		handler string
		args    []string
	)

	if key, handler, args, err = getHandlerParts(handlerKey); err != nil {
		return
	}

	var plugin Plugin
	if plugin, err = p.Get(key); err != nil {
		return
	}

	reflected := reflect.ValueOf(plugin).MethodByName(handler)
	if reflected.Kind() == reflect.Invalid {
		err = fmt.Errorf("method of <%s> not found within plugin <%s>", handler, key)
		return
	}

	toAssert := reflected.Interface()

	switch val := toAssert.(type) {
	case func(*httpserve.Context):
		h = val
		return

	case func(args ...string) (httpserve.Handler, error):
		return val(args...)

	default:
		err = fmt.Errorf("invalid handler type, expected Handler and received %T", val)
		return
	}
}

func canSet(a, b reflect.Value) (err error) {
	switch {
	// Check to see if the types match exactly
	case a.Type() == b.Type():
	// Check to see if the backend type implements the provided interface
	case a.Kind() == reflect.Interface && b.Type().Implements(a.Type()):

	default:
		// The provided value isn't an exact match, nor does it match the provided interface
		return fmt.Errorf("invalid type, expected %v and received %v", a.Type(), b.Type())
	}

	return
}
