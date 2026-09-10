package vroomy

import (
	"fmt"
	"strconv"
	"time"
)

// Environment holds string configuration values shared with plugin lifecycle hooks.
type Environment map[string]string

// Get returns the value for key, or an empty string if key is absent.
func (e Environment) Get(key string) (out string) {
	return e[key]
}

// GetInt parses a decimal int, returning zero and nil if key is absent.
func (e Environment) GetInt(key string) (out int, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return strconv.Atoi(val)
}

// GetInt64 parses a decimal int64, returning zero and nil if key is absent.
func (e Environment) GetInt64(key string) (out int64, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return strconv.ParseInt(val, 10, 64)
}

// GetFloat64 parses a float64, returning zero and nil if key is absent.
func (e Environment) GetFloat64(key string) (out float64, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return strconv.ParseFloat(val, 64)
}

// GetTime parses a time using layout, returning the zero time and nil if key is absent.
func (e Environment) GetTime(key, layout string) (out time.Time, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return time.Parse(layout, val)
}

// GetTimeInLocation parses a time using layout and loc, returning the zero time
// and nil if key is absent.
func (e Environment) GetTimeInLocation(key, layout string, loc *time.Location) (out time.Time, err error) {
	val, ok := e[key]
	if !ok {
		return
	}

	return time.ParseInLocation(layout, val, loc)
}

// Must returns an error if key is absent. A present empty string is accepted;
// despite its name, this method does not panic.
func (e Environment) Must(key string) (out string, err error) {
	var ok bool
	if out, ok = e[key]; !ok {
		err = fmt.Errorf("invalid environment value for <%s>, cannot be empty", key)
		return
	}

	return
}

// MustInt requires key to be present and parses its value as a decimal int.
func (e Environment) MustInt(key string) (out int, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return strconv.Atoi(val)
}

// MustInt64 requires key to be present and parses its value as a decimal int64.
func (e Environment) MustInt64(key string) (out int64, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return strconv.ParseInt(val, 10, 64)
}

// MustFloat64 requires key to be present and parses its value as a float64.
func (e Environment) MustFloat64(key string) (out float64, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return strconv.ParseFloat(val, 64)
}

// MustTime requires key to be present and parses its value using layout.
func (e Environment) MustTime(key, layout string) (out time.Time, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return time.Parse(layout, val)
}

// MustTimeInLocation requires key to be present and parses its value using layout
// and loc.
func (e Environment) MustTimeInLocation(key, layout string, loc *time.Location) (out time.Time, err error) {
	var val string
	if val, err = e.Must(key); err != nil {
		return
	}

	return time.ParseInLocation(layout, val, loc)
}
