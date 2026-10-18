package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const (
	configOptionPath     = "path"
	configOptionReadOnly = "readonly"
)

// SettableKeys returns dot-separated configuration paths using the exact field
// names exposed by the JSON schema.
func SettableKeys() []string {
	var keys []string
	collectSettableKeys(reflect.TypeOf(Config{}), nil, &keys)
	sort.Strings(keys)
	return keys
}

func collectSettableKeys(configType reflect.Type, prefix []string, keys *[]string) {
	for index := 0; index < configType.NumField(); index++ {
		field := configType.Field(index)
		name, ok := jsonFieldName(field)
		if !ok || hasConfigOption(field, configOptionReadOnly) {
			continue
		}
		path := appendPath(prefix, name)
		if field.Type.Kind() == reflect.Struct {
			collectSettableKeys(field.Type, path, keys)
			continue
		}
		*keys = append(*keys, strings.Join(path, "."))
	}
}

// SetValue updates a settable configuration field identified by its JSON path.
// Scalar values are parsed according to the destination Go type.
func SetValue(cfg *Config, key, value string) error {
	if cfg == nil {
		return fmt.Errorf("config must not be nil")
	}
	segments := strings.Split(key, ".")
	if key == "" || slicesContainEmpty(segments) {
		return unknownKeyError(key)
	}

	original := *cfg
	field, structField, ok := findSettableField(reflect.ValueOf(cfg).Elem(), segments)
	if !ok {
		return unknownKeyError(key)
	}
	if err := setScalarValue(field, structField, value); err != nil {
		return fmt.Errorf("set config key %q: %w", key, err)
	}
	if err := ValidateFields(*cfg, key); err != nil {
		*cfg = original
		return err
	}
	return nil
}

func findSettableField(value reflect.Value, segments []string) (reflect.Value, reflect.StructField, bool) {
	for segmentIndex, segment := range segments {
		field, structField, ok := fieldByJSONName(value, segment)
		if !ok || hasConfigOption(structField, configOptionReadOnly) {
			return reflect.Value{}, reflect.StructField{}, false
		}
		if segmentIndex == len(segments)-1 {
			if field.Kind() == reflect.Struct || !field.CanSet() {
				return reflect.Value{}, reflect.StructField{}, false
			}
			return field, structField, true
		}
		if field.Kind() != reflect.Struct {
			return reflect.Value{}, reflect.StructField{}, false
		}
		value = field
	}
	return reflect.Value{}, reflect.StructField{}, false
}

func fieldByJSONName(value reflect.Value, name string) (reflect.Value, reflect.StructField, bool) {
	valueType := value.Type()
	for index := 0; index < valueType.NumField(); index++ {
		structField := valueType.Field(index)
		fieldName, ok := jsonFieldName(structField)
		if ok && fieldName == name {
			return value.Field(index), structField, true
		}
	}
	return reflect.Value{}, reflect.StructField{}, false
}

func setScalarValue(field reflect.Value, structField reflect.StructField, value string) error {
	switch field.Kind() {
	case reflect.String:
		if value != "" && hasConfigOption(structField, configOptionPath) {
			value = filepath.Clean(value)
		}
		field.SetString(value)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse boolean value %q: %w", value, err)
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse integer value %q: %w", value, err)
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse unsigned integer value %q: %w", value, err)
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse numeric value %q: %w", value, err)
		}
		field.SetFloat(parsed)
	default:
		return fmt.Errorf("unsupported value type %s", field.Type())
	}
	return nil
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "" {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	return name, name != "" && name != "-"
}

func hasConfigOption(field reflect.StructField, option string) bool {
	for _, candidate := range strings.Split(field.Tag.Get("config"), ",") {
		if candidate == option {
			return true
		}
	}
	return false
}

func appendPath(prefix []string, name string) []string {
	path := make([]string, len(prefix), len(prefix)+1)
	copy(path, prefix)
	return append(path, name)
}

func slicesContainEmpty(values []string) bool {
	for _, value := range values {
		if value == "" {
			return true
		}
	}
	return false
}

func unknownKeyError(key string) error {
	return fmt.Errorf("unknown config key %q (valid keys: %v)", key, SettableKeys())
}
