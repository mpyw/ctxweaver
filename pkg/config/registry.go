package config

import (
	"maps"
	"slices"
)

// CarrierRegistry holds all registered carriers for quick lookup.
type CarrierRegistry struct {
	carriers map[string]CarrierDef // key: "package.Type"
}

// NewCarrierRegistry creates a registry, optionally loading default carriers.
func NewCarrierRegistry(includeDefaults bool) *CarrierRegistry {
	r := &CarrierRegistry{
		carriers: make(map[string]CarrierDef),
	}
	if includeDefaults {
		for _, c := range defaultCarriers {
			r.Register(c)
		}
	}
	return r
}

// Register adds a carrier to the registry.
func (r *CarrierRegistry) Register(c CarrierDef) {
	key := c.Package + "." + c.Type
	r.carriers[key] = c
}

// Lookup finds a carrier by package path and type name.
func (r *CarrierRegistry) Lookup(packagePath, typeName string) (CarrierDef, bool) {
	key := packagePath + "." + typeName
	c, ok := r.carriers[key]
	return c, ok
}

// All returns all registered carriers, ordered by their "package.Type" key
// so that callers observe a stable sequence.
func (r *CarrierRegistry) All() []CarrierDef {
	keys := slices.Sorted(maps.Keys(r.carriers))
	result := make([]CarrierDef, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.carriers[key])
	}
	return result
}
