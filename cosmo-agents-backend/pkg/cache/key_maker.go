package cache

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

// KeyMaker defines the interface for generating cache keys.
type KeyMaker interface {
	// Make generates a cache key based on function name, prefix, and arguments.
	Make(fn interface{}, prefix string, args ...interface{}) string
}

// DefaultKeyMaker implements the KeyMaker interface with default behavior.
type DefaultKeyMaker struct{}

// NewDefaultKeyMaker creates a new DefaultKeyMaker.
func NewDefaultKeyMaker() *DefaultKeyMaker {
	return &DefaultKeyMaker{}
}

// Make generates a cache key based on function name, prefix, and arguments.
func (km *DefaultKeyMaker) Make(fn interface{}, prefix string, args ...interface{}) string {
	// Get function name and package
	fnValue := reflect.ValueOf(fn)
	fnPtr := fnValue.Pointer()
	fnInfo := runtime.FuncForPC(fnPtr)

	var functionName string
	if fnInfo != nil {
		functionName = fnInfo.Name()
	} else {
		functionName = "unknown_function"
	}

	// Build base path
	path := fmt.Sprintf("%s:%s", prefix, functionName)

	// Serialize arguments
	if len(args) > 0 {
		params := make([]string, 0, len(args))
		for i, arg := range args {
			serialized := km.serialize(arg)
			params = append(params, fmt.Sprintf("arg%d=%s", i, serialized))
		}
		paramsStr := strings.Join(params, ":")
		return fmt.Sprintf("%s:%s", path, paramsStr)
	}

	return path
}

// serialize converts a value to a string representation.
func (km *DefaultKeyMaker) serialize(value interface{}) string {
	if value == nil {
		return "nil"
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", v.Float())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Slice, reflect.Array:
		parts := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			parts[i] = km.serialize(v.Index(i).Interface())
		}
		return strings.Join(parts, ",")
	case reflect.Map:
		keys := v.MapKeys()
		// Sort keys for consistent ordering
		keyStrs := make([]string, len(keys))
		for i, k := range keys {
			keyStrs[i] = km.serialize(k.Interface())
		}
		sort.Strings(keyStrs)

		pairs := make([]string, 0, len(keys))
		for _, keyStr := range keyStrs {
			for _, k := range keys {
				if km.serialize(k.Interface()) == keyStr {
					valStr := km.serialize(v.MapIndex(k).Interface())
					pairs = append(pairs, fmt.Sprintf("%s=%s", keyStr, valStr))
					break
				}
			}
		}
		return strings.Join(pairs, ",")
	case reflect.Struct:
		t := v.Type()
		fields := make([]string, 0, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.IsExported() {
				fieldValue := v.Field(i)
				fields = append(fields, fmt.Sprintf("%s=%s", field.Name, km.serialize(fieldValue.Interface())))
			}
		}
		return strings.Join(fields, ",")
	default:
		return fmt.Sprintf("%v", value)
	}
}
