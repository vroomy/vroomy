package vroomy

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/vroomy/httpserve"
	"golang.org/x/crypto/acme/autocert"
)

type listener interface {
	Listen(port uint16) error
}

func initDir(loc string) (err error) {
	if len(loc) == 0 {
		return err
	}

	if err = os.Mkdir(loc, 0744); err == nil {
		return err
	}

	if os.IsExist(err) {
		return nil
	}

	return err
}

func getHandlerParts(handlerKey string) (key, handler string, args []string, err error) {
	spl := strings.SplitN(handlerKey, ".", 2)
	if len(spl) != 2 {
		err = fmt.Errorf("expected key and handler, received \"%s\"", handlerKey)
		return key, handler, args, err
	}

	key = spl[0]
	handler = spl[1]

	spl = strings.Split(handler, "(")
	if len(spl) == 1 {
		return key, handler, args, err
	}

	handler = spl[0]
	argsStr := spl[1]

	if len(argsStr) == 0 || argsStr[len(argsStr)-1] != ')' {
		err = ErrExpectedEndParen
		return key, handler, args, err
	}

	argsStr = argsStr[:len(argsStr)-1]
	args = strings.Split(argsStr, ",")
	return key, handler, args, err
}

func getHostPolicy() (hp autocert.HostPolicy, err error) {
	var method interface{}
	method, err = getPluginMethod("autocert", "HostPolicy")
	switch {
	case err == nil:
		return assertAsHostPolicy(method)
	case isUnregisteredPluginError(err):
		return nil, nil
	default:
		return hp, err
	}
}

func getPluginMethod(pluginKey, method string) (out interface{}, err error) {
	var plugin Plugin
	if plugin, err = p.Get(pluginKey); err != nil {
		return out, err
	}

	reflected := reflect.ValueOf(plugin).MethodByName(method)
	if reflected.Kind() == reflect.Invalid {
		err = fmt.Errorf("method of <%s> not found within plugin <%s>", method, pluginKey)
		return out, err
	}

	out = reflected.Interface()
	return out, err
}

func getHandler(handlerKey string) (h httpserve.Handler, err error) {
	var (
		key     string
		handler string
		args    []string
	)

	if key, handler, args, err = getHandlerParts(handlerKey); err != nil {
		return h, err
	}

	var toAssert interface{}
	if toAssert, err = getPluginMethod(key, handler); err != nil {
		return h, err
	}

	switch val := toAssert.(type) {
	case func(*httpserve.Context):
		h = val
		return h, err

	case func(args ...string) (httpserve.Handler, error):
		return val(args...)

	default:
		err = fmt.Errorf("invalid handler type, expected Handler and received %T", val)
		return h, err
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

	return err
}

func copySlice[T any](in []T) (out []T) {
	out = make([]T, len(in))
	copy(out, in)
	return out
}

func getField(rval reflect.Value, indices []int) (field reflect.Value) {
	for _, index := range indices {
		if rval.Kind() == reflect.Ptr {
			rval = reflect.Indirect(rval.Elem())
		}

		rval = rval.Field(index)
	}

	if rval.CanAddr() {
		rval = rval.Addr()
	}

	return rval
}

func isUnregisteredPluginError(err error) bool {
	str := err.Error()
	switch {
	case !strings.Contains(str, "plugin with key of <"):
		return false
	case !strings.Contains(str, "> has not been registered"):
		return false
	default:
		return true
	}
}

func assertAsHostPolicy(fn interface{}) (hp autocert.HostPolicy, err error) {
	var ok bool
	hp, ok = fn.(func(ctx context.Context, host string) error)
	if !ok {
		err = ErrInvalidHostPolicy
		return hp, err
	}

	return hp, err
}
