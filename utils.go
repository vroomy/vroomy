package vroomy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/vroomy/httpserve"
	"golang.org/x/crypto/acme/autocert"
)

func initDir(loc string) (err error) {
	if len(loc) == 0 {
		return nil
	}

	if err = os.Mkdir(loc, 0744); err == nil {
		return nil
	}

	if os.IsExist(err) {
		var info os.FileInfo
		if info, err = os.Stat(loc); err != nil {
			return err
		}

		if !info.IsDir() {
			return fmt.Errorf("data path %q is not a directory", loc)
		}

		return nil
	}

	return err
}

func getHandlerParts(handlerKey string) (key, handler string, args []string, err error) {
	spl := strings.SplitN(handlerKey, ".", 2)
	if len(spl) != 2 || spl[0] == "" || spl[1] == "" {
		err = fmt.Errorf("expected key and handler, received \"%s\"", handlerKey)
		return key, handler, args, err
	}

	key = spl[0]
	handler = spl[1]

	spl = strings.Split(handler, "(")
	if len(spl) == 1 {
		return key, handler, args, nil
	}

	handler = spl[0]
	argsStr := spl[1]

	if len(argsStr) == 0 || argsStr[len(argsStr)-1] != ')' {
		err = ErrExpectedEndParen
		return key, handler, args, err
	}

	argsStr = argsStr[:len(argsStr)-1]
	args = strings.Split(argsStr, ",")
	return key, handler, args, nil
}

func getHostPolicy() (hp autocert.HostPolicy, err error) {
	var method interface{}
	method, err = getPluginMethod("autocert", "HostPolicy")
	switch {
	case err == nil:
		return assertAsHostPolicy(method)
	case errors.Is(err, ErrPluginNotLoaded):
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

	if err = validatePlugin(plugin); err != nil {
		return nil, fmt.Errorf("get method from plugin %q: %w", pluginKey, err)
	}

	reflected := reflect.ValueOf(plugin).MethodByName(method)
	if reflected.Kind() == reflect.Invalid {
		err = fmt.Errorf("method of <%s> not found within plugin <%s>", method, pluginKey)
		return out, err
	}

	out = reflected.Interface()
	return out, nil
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
		return h, nil

	case func(args ...string) (httpserve.Handler, error):
		if h, err = val(args...); err != nil {
			return nil, fmt.Errorf("create handler %s.%s: %w", key, handler, err)
		}

		if h == nil {
			return nil, fmt.Errorf("handler factory %s.%s returned nil: %w", key, handler, ErrInvalidPluginHandler)
		}

		return h, nil

	default:
		err = fmt.Errorf("invalid handler type, expected Handler and received %T: %w", val, ErrInvalidPluginHandler)
		return h, err
	}
}

func canSet(a, b reflect.Value) (err error) {
	if !a.IsValid() || !b.IsValid() {
		return fmt.Errorf("invalid dependency value")
	}

	switch {
	// Check to see if the types match exactly
	case a.Type() == b.Type():
	// Check to see if the backend type implements the provided interface
	case a.Kind() == reflect.Interface && b.Type().Implements(a.Type()):

	default:
		// The provided value isn't an exact match, nor does it match the provided interface
		return fmt.Errorf("invalid type, expected %v and received %v", a.Type(), b.Type())
	}

	return nil
}

func copySlice[T any](in []T) (out []T) {
	out = make([]T, len(in))
	copy(out, in)
	return out
}

func getField(value reflect.Value, indices []int) (field reflect.Value, err error) {
	for _, index := range indices {
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return field, fmt.Errorf("nil anonymous pointer in dependency path")
			}

			value = value.Elem()
		}

		if !value.IsValid() || value.Kind() != reflect.Struct || index < 0 || index >= value.NumField() {
			return field, fmt.Errorf("invalid dependency field path")
		}

		value = value.Field(index)
	}

	if !value.IsValid() || !value.CanSet() {
		return field, ErrNotAddressable
	}

	return value, nil
}

func isNilValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}

	return false
}

func assertAsHostPolicy(fn interface{}) (hp autocert.HostPolicy, err error) {
	var ok bool
	hp, ok = fn.(func(ctx context.Context, host string) error)
	if !ok {
		err = ErrInvalidHostPolicy
		return hp, err
	}

	return hp, nil
}

// withoutCancellation preserves each leaf error except normal context cancellation.
func withoutCancellation(err error) error {
	if err == nil || err == context.Canceled {
		return nil
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		var remaining []error
		for _, cause := range joined.Unwrap() {
			remaining = append(remaining, withoutCancellation(cause))
		}

		return errors.Join(remaining...)
	}

	return err
}
