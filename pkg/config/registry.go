package config

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

// All returns all registered carriers.
func (r *CarrierRegistry) All() []CarrierDef {
	result := make([]CarrierDef, 0, len(r.carriers))
	for _, c := range r.carriers {
		result = append(result, c)
	}
	return result
}
