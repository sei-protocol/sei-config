package seiconfig

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

const (
	envPrefix    = "SEI_"
	legacyPrefix = "SEID_"
)

// ResolveEnv applies environment variable overrides to cfg using the
// SEI_<SECTION>_<FIELD> naming convention. During the transition period,
// SEID_ prefixed vars are also recognized (with lower precedence).
// Returns a list of deprecation warnings for any SEID_ vars that were used.
func ResolveEnv(cfg *SeiConfig) []string {
	var warnings []string

	// Build the env var map from struct tags
	envMap := buildEnvMap(cfg)

	for envVar, fieldPath := range envMap {
		seiVar := envPrefix + envVar
		seidVar := legacyPrefix + envVar

		var value string
		var found bool

		if v, ok := os.LookupEnv(seiVar); ok {
			value = v
			found = true
		}

		if v, ok := os.LookupEnv(seidVar); ok {
			if !found {
				value = v
				found = true
				warnings = append(warnings, fmt.Sprintf(
					"environment variable %s is deprecated; use %s instead", seidVar, seiVar))
			}
		}

		if found {
			if err := setFieldByPath(cfg, fieldPath, value); err != nil {
				warnings = append(warnings, fmt.Sprintf(
					"failed to apply env var %s: %v", seiVar, err))
			}
		}
	}

	return warnings
}

// buildEnvMap returns a mapping from env var suffix (without prefix) to the
// struct field path. For example, "CHAIN_MIN_GAS_PRICES" -> "Chain.MinGasPrices".
func buildEnvMap(_ *SeiConfig) map[string]string {
	result := make(map[string]string)
	buildEnvMapRecursive(reflect.TypeFor[SeiConfig](), "", "", result)
	return result
}

func buildEnvMapRecursive(t reflect.Type, envPrefix, fieldPrefix string, result map[string]string) {
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tomlTag := field.Tag.Get("toml")
		if tomlTag == "" || tomlTag == "-" {
			continue
		}

		envKey := strings.ToUpper(tomlTag)
		if envPrefix != "" {
			envKey = envPrefix + "_" + envKey
		}

		fieldPath := field.Name
		if fieldPrefix != "" {
			fieldPath = fieldPrefix + "." + field.Name
		}

		ft := field.Type
		if ft.Kind() == reflect.Struct && ft != reflect.TypeFor[Duration]() {
			buildEnvMapRecursive(ft, envKey, fieldPath, result)
		} else {
			result[envKey] = fieldPath
		}
	}
}

// setFieldByPath sets a struct field by its dot-separated path using reflection.
func setFieldByPath(cfg *SeiConfig, path string, value string) error {
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(cfg).Elem()

	for _, part := range parts {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return fmt.Errorf("nil pointer at %s", part)
			}
			v = v.Elem()
		}
		v = v.FieldByName(part)
		if !v.IsValid() {
			return fmt.Errorf("field %s not found", part)
		}
	}

	if !v.CanSet() {
		return fmt.Errorf("field %s is not settable", path)
	}

	return setReflectValue(v, value)
}

func setReflectValue(v reflect.Value, s string) error {
	if v.Type() == reflect.TypeFor[Duration]() {
		var d Duration
		if err := d.UnmarshalText([]byte(s)); err != nil {
			return err
		}
		v.Set(reflect.ValueOf(d))
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		switch strings.ToLower(s) {
		case "true", "1", "yes":
			v.SetBool(true)
		case "false", "0", "no":
			v.SetBool(false)
		default:
			return fmt.Errorf("invalid bool value: %q", s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := parseInt64(s)
		if err != nil {
			return err
		}
		if v.OverflowInt(n) {
			return fmt.Errorf("value %d overflows %s", n, v.Type())
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := parseUint64(s)
		if err != nil {
			return err
		}
		if v.OverflowUint(n) {
			return fmt.Errorf("value %d overflows %s", n, v.Type())
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := parseFloat64(s)
		if err != nil {
			return err
		}
		if v.OverflowFloat(n) {
			return fmt.Errorf("value %g overflows %s", n, v.Type())
		}
		v.SetFloat(n)
	default:
		return fmt.Errorf("unsupported field type: %s", v.Type())
	}
	return nil
}

func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func parseUint64(s string) (uint64, error) {
	var n uint64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func parseFloat64(s string) (float64, error) {
	var n float64
	_, err := fmt.Sscanf(s, "%f", &n)
	return n, err
}
