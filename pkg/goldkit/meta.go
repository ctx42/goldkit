package goldkit

import (
	"errors"
	"fmt"
	"time"
)

// Golden file metadata errors.
var (
	// ErrMissing represents an error for a missing golden file metadata key.
	ErrMissing = errors.New("missing metadata key")

	// ErrType represents an error for invalid golden file metadata key.
	ErrType = errors.New("invalid metadata key type")

	// ErrFormat represents an error for invalid golden file metadata key.
	ErrFormat = errors.New("invalid metadata key format")

	// ErrValue represents an error for invalid golden file metadata key.
	ErrValue = errors.New("invalid metadata key value")
)

// Meta is a helper type for manipulating metadata used by templates and golden
// files.
type Meta map[string]any

// metaContract pins the exported method set of [Meta]. The `Meta` prefix on
// every method is deliberate and load-bearing, not a stutter to be cleaned up:
// these names form a contract relied on outside this package. Golden files
// invoke the methods by name through [text/template], and external consumers
// match [Meta] structurally against their own metadata interface, which this
// package purposely does not import to avoid the dependency. Renaming a method
// or changing its signature would silently break those callers with no local
// compile error, so the assertion below turns any such change into a build
// failure here instead. Keep this interface in sync with [Meta]'s methods; do
// not rename them.
type metaContract interface {
	MetaLookup(key string) (any, bool)
	MetaGet(key string) any
	MetaSet(key string, value any) Meta
	MetaDelete(key string)
	MetaMeta(key string) (Meta, error)
	MetaGetString(key string) (string, error)
	MetaGetBool(key string) (bool, error)
	MetaGetInt(key string) (int, error)
	MetaGetInt64(key string) (int64, error)
	MetaGetFloat64(key string) (float64, error)
	MetaGetTime(key string) (time.Time, error)
	MetaGetTimeIn(key string, tz *time.Location) (time.Time, error)
	MetaGetLoc(key string) (*time.Location, error)
}

var _ metaContract = Meta(nil) // Never rename or re-sign [Meta] methods.

// MetaLookup returns the value of the map variable named by the key. If the
// variable is present in the map collection, the value (which may be empty or
// nil) is returned and the boolean is true. Otherwise, the returned value will
// be nil and the boolean will be false.
func (m Meta) MetaLookup(key string) (any, bool) {
	val, ok := m[key]
	return val, ok
}

// MetaGet returns the value of the map variable named by the key. If the
// variable is not present in the map nil is returned.
func (m Meta) MetaGet(key string) any { return m[key] }

// MetaSet sets the value of the map named by the key and returns the receiver
// for chaining. The receiver must be non-nil; like any write to a nil map,
// calling MetaSet on a nil [Meta] panics.
func (m Meta) MetaSet(key string, value any) Meta {
	m[key] = value
	return m
}

// MetaDelete deletes the map entry identified by the key.
func (m Meta) MetaDelete(key string) { delete(m, key) }

// MetaMeta checks if the specified key exists in the map, and it is of
// map[string]any or [Meta] type. If the key is missing, it returns nil, and
// the error has [ErrMissing] in its chain. If the key exists but its value is
// not of the expected type, it returns nil and error having [ErrType] in its
// chain. Otherwise, it returns the [Meta] value of the key and a nil error.
func (m Meta) MetaMeta(key string) (Meta, error) {
	if val, ok := m[key]; ok {
		if sub, ok := val.(map[string]any); ok {
			return sub, nil
		}
		if sub, ok := val.(Meta); ok {
			return sub, nil
		}
		return nil, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return nil, fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetString checks if the specified key exists in the map, and it is of
// string type. If the key is missing, it returns an empty string and error
// having [ErrMissing] in its chain. If the key exists but its value is not of
// the expected type, it returns an empty string and error having [ErrType] in
// its chain. Otherwise, it returns the string value of the key and a nil error.
func (m Meta) MetaGetString(key string) (string, error) {
	if val, ok := m[key]; ok {
		if v, ok := val.(string); ok {
			return v, nil
		}
		return "", fmt.Errorf("%w: %#q", ErrType, key)
	}
	return "", fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetBool checks if the specified key exists in the map, and it is of the
// bool type. If the key is missing, it returns false, and the error has
// [ErrMissing] in its chain. If the key exists but its value is not of the
// expected type, it returns false and error having [ErrType] in its chain.
// Otherwise, it returns the boolean value of the key and a nil error.
func (m Meta) MetaGetBool(key string) (bool, error) {
	if val, ok := m[key]; ok {
		if v, ok := val.(bool); ok {
			return v, nil
		}
		return false, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return false, fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetInt checks if the specified key exists in the map, and it is of int
// type. If the key is missing, it returns 0, and the error has [ErrMissing] in
// its chain. If the key exists but its value is not of the expected type, it
// returns 0 and error having [ErrType] in its chain. Otherwise, it returns the
// int value of the key and a nil error.
func (m Meta) MetaGetInt(key string) (int, error) {
	if val, ok := m[key]; ok {
		if v, ok := val.(int); ok {
			return v, nil
		}
		return 0, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return 0, fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetInt64 checks if the specified key exists in the map, and it is one of
// the int, int8, int16, int32, int64 types. If the key is missing, it returns
// 0, and the error has [ErrMissing] in its chain. If the key exists but its
// value is not of expected types, it returns 0 and error having [ErrType] in
// its chain. Otherwise, it returns the int64 value of the key and a nil error.
func (m Meta) MetaGetInt64(key string) (int64, error) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return int64(v), nil
		case int8:
			return int64(v), nil
		case int16:
			return int64(v), nil
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		}
		return 0, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return 0, fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetFloat64 checks if the specified key exists in the map, and it is one
// of the int, int8, int16, int32, int64, float32, float64 types. If the key is
// missing, it returns 0.0, and the error has [ErrMissing] in its chain. If the
// key exists but its value is not of expected types, it returns 0.0 and error
// having [ErrType] in its chain. Otherwise, it returns the float64 value of
// the key and a nil error.
func (m Meta) MetaGetFloat64(key string) (float64, error) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return float64(v), nil
		case int8:
			return float64(v), nil
		case int16:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case float32:
			return float64(v), nil
		case float64:
			return v, nil
		}
		return 0, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return 0, fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetTime checks if the specified key exists in the map, and it is of
// [time.Time] type or string representing time in [time.RFC3339] format. If
// the key is missing, it returns zero value time and error having [ErrMissing]
// in its chain. If the key exists but its value is not of the expected type,
// it returns zero value time and error having [ErrType] in its chain. If the
// key is a string, but it is not [time.RFC3339] or special case of zero value
// time as a string, it returns zero value time and error having [ErrFormat] in
// its chain. Otherwise, it returns the [time.Time] value of the key and a nil
// error.
//
// The special case of "0000-00-00T00:00:00" is also handled for which the
// zero-value time is returned and nil error.
func (m Meta) MetaGetTime(key string) (time.Time, error) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case time.Time:
			return v, nil
		case string:
			// Zero value time.
			if v == "0000-00-00T00:00:00" {
				return time.Time{}, nil
			}
			tim, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return time.Time{}, fmt.Errorf("%w: %#q", ErrFormat, key)
			}
			return tim, nil
		}
		return time.Time{}, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return time.Time{}, fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetTimeIn checks if the specified key exists in the map, and it is of
// [time.Time] type or string representing time in "2006-01-02T15:04:05" format.
// If the key is missing, it returns zero value time and error having
// [ErrMissing] in its chain. If the key exists but its value is not of the
// expected type, it returns zero value time and error having [ErrType] in its
// chain. If the key is a string, but it is not [time.RFC3339] or special case
// of zero value time as a string, it returns zero value time and error having
// [ErrFormat] in its chain. Otherwise, it returns the [time.Time] value of the
// key in a given timezone and a nil error.
//
// The special case of "0000-00-00T00:00:00" is also handled for which the
// zero-value time is returned and nil error.
func (m Meta) MetaGetTimeIn(key string, tz *time.Location) (time.Time, error) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case time.Time:
			return v, nil
		case string:
			// Zero value time.
			if v == "0000-00-00T00:00:00" {
				return time.Time{}, nil
			}
			// Try RFC3339.
			tim, err := time.Parse(time.RFC3339, v)
			if err != nil {
				// Try RFC3339 without timezone information.
				const format = "2006-01-02T15:04:05"
				if tim, err = time.ParseInLocation(format, v, tz); err != nil {
					return time.Time{}, fmt.Errorf("%w: %#q", ErrFormat, key)
				}
				return tim, nil
			}
			return tim.In(tz), nil
		}
		return time.Time{}, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return time.Time{}, fmt.Errorf("%w: %#q", ErrMissing, key)
}

// MetaGetLoc checks if the specified key exists in the map, and it is a string
// timezone name (e.g., Europe/Warsaw). If the key is missing, it returns nil,
// and the error has [ErrMissing] in its chain. If the key exists but its value
// is not of the expected type, it returns nil and error having [ErrType] in
// its chain. If the key is a string, but it is not a timezone name, it returns
// nil and an error having [ErrFormat] in its chain. Otherwise, it returns the
// [time.Location] value of the key and a nil error.
func (m Meta) MetaGetLoc(key string) (*time.Location, error) {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case *time.Location:
			return v, nil
		case string:
			loc, err := time.LoadLocation(v)
			if err != nil {
				return nil, fmt.Errorf("%w: %#q", ErrFormat, key)
			}
			return loc, nil
		}
		return nil, fmt.Errorf("%w: %#q", ErrType, key)
	}
	return nil, fmt.Errorf("%w: %#q", ErrMissing, key)
}
