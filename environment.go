package vroomy

import (
	"fmt"
	"strconv"
	"time"
)

type Environment map[string]string

func (e Environment) Get(key string) (out string) {
	return e[key]
}

func (e Environment) GetInt(key string) (out int, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return strconv.Atoi(val)
}

func (e Environment) GetInt64(key string) (out int64, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return strconv.ParseInt(val, 10, 64)
}

func (e Environment) GetFloat64(key string) (out float64, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return strconv.ParseFloat(val, 64)
}

func (e Environment) GetTime(key, layout string) (out time.Time, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return time.Parse(layout, val)
}

func (e Environment) GetTimeInLocation(key, layout string, loc *time.Location) (out time.Time, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return time.ParseInLocation(layout, val, loc)
}

func (e Environment) Must(key string) (out string, err error) {
	var ok bool
	if out, ok = e[key]; !ok {
		err = fmt.Errorf("invalid environment value for <%s>, cannot be empty", key)
		return
	}

	return
}

func (e Environment) MustInt(key string) (out int, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return strconv.Atoi(val)
}

func (e Environment) MustInt64(key string) (out int64, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return strconv.ParseInt(val, 10, 64)
}

func (e Environment) MustFloat64(key string) (out float64, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return strconv.ParseFloat(val, 64)
}

func (e Environment) MustTime(key, layout string) (out time.Time, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return time.Parse(layout, val)
}

func (e Environment) MustTimeInLocation(key, layout string, loc *time.Location) (out time.Time, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return time.ParseInLocation(layout, val, loc)
}
