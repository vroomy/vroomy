package vroomy

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/hatchify/errors"
	"github.com/vroomy/common"
	"github.com/vroomy/plugins"
)

const (
	// ErrExpectedEndParen is returned when an ending parenthesis is missing
	ErrExpectedEndParen = errors.Error("expected ending parenthesis")
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

func getHandler(handlerKey string) (h common.Handler, err error) {
	var (
		key     string
		handler string
		args    []string
	)

	if key, handler, args, err = getHandlerParts(handlerKey); err != nil {
		return
	}

	var p plugins.Plugin
	if p, err = plugins.Get(key); err != nil {
		return
	}

	reflected := reflect.ValueOf(p).MethodByName(handler)
	if reflected.Kind() == reflect.Invalid {
		err = fmt.Errorf("method of <%s> not found within plugin <%s>", handler, key)
		return
	}

	toAssert := reflected.Interface()

	switch val := toAssert.(type) {
	case func(common.Context):
		h = val
		return

	case func(args ...string) (common.Handler, error):
		return val(args...)

	default:
		err = fmt.Errorf("invalid handler type, expected common.Handler and received %T", val)
		return
	}
}
