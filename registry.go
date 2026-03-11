package seiconfig

import (
	"reflect"
	"sort"
	"strings"
)

// FieldType classifies the Go type of a config field for consumers that
// need to present or validate values without full Go type information
// (e.g. CRD validation webhooks, CLI flag generation, documentation).
type FieldType int

const (
	FieldTypeString FieldType = iota
	FieldTypeInt              // int, int64
	FieldTypeUint             // uint, uint16, uint32, uint64
	FieldTypeFloat            // float64
	FieldTypeBool
	FieldTypeDuration    // Duration (encoded as string in TOML)
	FieldTypeStringSlice // []string
	FieldTypeOther       // anything else (nested labels, etc.)
)

func (ft FieldType) String() string {
	switch ft {
	case FieldTypeString:
		return "string"
	case FieldTypeInt:
		return "int"
	case FieldTypeUint:
		return "uint"
	case FieldTypeFloat:
		return "float"
	case FieldTypeBool:
		return "bool"
	case FieldTypeDuration:
		return "duration"
	case FieldTypeStringSlice:
		return "[]string"
	default:
		return "other"
	}
}

// ConfigField describes a single configuration parameter. Fields are
// auto-discovered from SeiConfig's struct tags and enriched with
// hand-authored metadata via Enrich().
type ConfigField struct {
	// Key is the dotted TOML key path in the unified schema.
	// Example: "evm.http_port", "storage.state_store.keep_recent"
	Key string

	// EnvVar is the environment variable name (SEI_ prefix).
	// Example: "SEI_EVM_HTTP_PORT"
	EnvVar string

	// FieldPath is the Go struct field path for reflection access.
	// Example: "EVM.HTTPPort"
	FieldPath string

	// Type classifies the field's Go type.
	Type FieldType

	// Description is a human-readable explanation of the field.
	Description string

	// Unit describes the value's unit when not obvious from the type.
	// Examples: "bytes", "blocks", "connections", "bytes/sec"
	Unit string

	// HotReload indicates the field can be changed at runtime without
	// restarting seid.
	HotReload bool

	// Deprecated indicates the field is scheduled for removal.
	Deprecated bool

	// Section is the top-level config section this field belongs to.
	// Example: "evm", "storage", "chain"
	Section string
}

// Registry holds metadata for every field in SeiConfig. It is built once
// via BuildRegistry() and is safe for concurrent read access.
type Registry struct {
	fields   []ConfigField
	byKey    map[string]*ConfigField
	byEnvVar map[string]*ConfigField
}

// BuildRegistry constructs a Registry by reflecting over SeiConfig's struct
// tags to auto-populate key paths, env var names, field types, and sections.
func BuildRegistry() *Registry {
	r := &Registry{
		byKey:    make(map[string]*ConfigField),
		byEnvVar: make(map[string]*ConfigField),
	}

	buildFieldsRecursive(
		reflect.TypeFor[SeiConfig](),
		"", // toml prefix
		"", // field path prefix
		"", // section
		r,
	)

	sort.Slice(r.fields, func(i, j int) bool {
		return r.fields[i].Key < r.fields[j].Key
	})

	// Rebuild index pointers after sort
	for i := range r.fields {
		f := &r.fields[i]
		r.byKey[f.Key] = f
		r.byEnvVar[f.EnvVar] = f
	}

	return r
}

func buildFieldsRecursive(
	t reflect.Type,
	tomlPrefix, fieldPrefix, section string,
	r *Registry,
) {
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		tag := sf.Tag.Get("toml")
		if tag == "" || tag == "-" {
			continue
		}

		tomlKey := tag
		if tomlPrefix != "" {
			tomlKey = tomlPrefix + "." + tag
		}

		fieldPath := sf.Name
		if fieldPrefix != "" {
			fieldPath = fieldPrefix + "." + sf.Name
		}

		currentSection := section
		if currentSection == "" {
			currentSection = tag
		}

		ft := sf.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if ft.Kind() == reflect.Struct && ft != reflect.TypeFor[Duration]() {
			buildFieldsRecursive(ft, tomlKey, fieldPath, currentSection, r)
			continue
		}

		envSuffix := strings.ToUpper(strings.ReplaceAll(tomlKey, ".", "_"))

		f := ConfigField{
			Key:       tomlKey,
			EnvVar:    "SEI_" + envSuffix,
			FieldPath: fieldPath,
			Type:      classifyType(ft),
			Section:   currentSection,
		}

		r.fields = append(r.fields, f)
	}
}

func classifyType(t reflect.Type) FieldType {
	if t == reflect.TypeFor[Duration]() {
		return FieldTypeDuration
	}
	if t == reflect.TypeFor[[]string]() {
		return FieldTypeStringSlice
	}

	switch t.Kind() {
	case reflect.String:
		return FieldTypeString
	case reflect.Bool:
		return FieldTypeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return FieldTypeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return FieldTypeUint
	case reflect.Float32, reflect.Float64:
		return FieldTypeFloat
	default:
		return FieldTypeOther
	}
}

// ---------------------------------------------------------------------------
// Query methods
// ---------------------------------------------------------------------------

// Fields returns all registered fields in key-sorted order.
func (r *Registry) Fields() []ConfigField {
	out := make([]ConfigField, len(r.fields))
	copy(out, r.fields)
	return out
}

// Field returns the metadata for a single field by its TOML key path,
// or nil if the key is not found.
func (r *Registry) Field(key string) *ConfigField {
	return r.byKey[key]
}

// FieldByEnvVar returns the metadata for a field by its environment
// variable name, or nil if not found.
func (r *Registry) FieldByEnvVar(envVar string) *ConfigField {
	return r.byEnvVar[envVar]
}

// FieldsInSection returns all fields belonging to a top-level section.
func (r *Registry) FieldsInSection(section string) []ConfigField {
	var out []ConfigField
	for _, f := range r.fields {
		if f.Section == section {
			out = append(out, f)
		}
	}
	return out
}

// HotReloadableFields returns all fields marked as safe to change at runtime.
func (r *Registry) HotReloadableFields() []ConfigField {
	var out []ConfigField
	for _, f := range r.fields {
		if f.HotReload {
			out = append(out, f)
		}
	}
	return out
}

// DeprecatedFields returns all fields marked as deprecated.
func (r *Registry) DeprecatedFields() []ConfigField {
	var out []ConfigField
	for _, f := range r.fields {
		if f.Deprecated {
			out = append(out, f)
		}
	}
	return out
}

// Sections returns the unique top-level section names in sorted order.
func (r *Registry) Sections() []string {
	seen := make(map[string]bool)
	for _, f := range r.fields {
		seen[f.Section] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// Len returns the total number of registered fields.
func (r *Registry) Len() int {
	return len(r.fields)
}

// DefaultsByMode returns the default value for every field under the given
// mode as a map from TOML key to its string representation. This is useful
// for documentation generation and diff tooling.
func (r *Registry) DefaultsByMode(mode NodeMode) map[string]any {
	cfg := DefaultForMode(mode)
	return r.extractValues(cfg)
}

func (r *Registry) extractValues(cfg *SeiConfig) map[string]any {
	result := make(map[string]any, len(r.fields))
	v := reflect.ValueOf(cfg).Elem()
	for _, f := range r.fields {
		val := navigateToField(v, f.FieldPath)
		if val.IsValid() {
			result[f.Key] = val.Interface()
		}
	}
	return result
}

func navigateToField(v reflect.Value, path string) reflect.Value {
	for part := range strings.SplitSeq(path, ".") {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.FieldByName(part)
		if !v.IsValid() {
			return v
		}
	}
	return v
}

// ---------------------------------------------------------------------------
// Enrichment
// ---------------------------------------------------------------------------

// FieldOption is a functional option for enriching a ConfigField.
type FieldOption func(*ConfigField)

// WithDescription sets the field's human-readable description.
func WithDescription(desc string) FieldOption {
	return func(f *ConfigField) { f.Description = desc }
}

// WithUnit sets the field's value unit label.
func WithUnit(unit string) FieldOption {
	return func(f *ConfigField) { f.Unit = unit }
}

// WithHotReload marks the field as safe to change without a restart.
func WithHotReload() FieldOption {
	return func(f *ConfigField) { f.HotReload = true }
}

// WithDeprecated marks the field as deprecated.
func WithDeprecated() FieldOption {
	return func(f *ConfigField) { f.Deprecated = true }
}

// Enrich updates a field's metadata by key. Returns false if the key
// was not found in the registry.
func (r *Registry) Enrich(key string, opts ...FieldOption) bool {
	f := r.byKey[key]
	if f == nil {
		return false
	}
	for _, opt := range opts {
		opt(f)
	}
	return true
}

// EnrichAll applies multiple enrichments at once. The map keys are TOML
// key paths. Returns the keys that were not found.
func (r *Registry) EnrichAll(enrichments map[string][]FieldOption) []string {
	var missing []string
	for key, opts := range enrichments {
		if !r.Enrich(key, opts...) {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}
