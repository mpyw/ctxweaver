package config

import (
	_ "embed"

	"gopkg.in/yaml.v3"

	"github.com/mpyw/ctxweaver/internal"
)

//go:embed carriers.yaml
var defaultCarriersYAML []byte

// defaultCarriers holds the built-in carrier definitions embedded from
// carriers.yaml. Parsed at init time - failure here means corrupted embedded
// files.
//
// Shared with registry.go: NewCarrierRegistry seeds registries from this list.
//
//declscope:package
var defaultCarriers []CarrierDef

func init() {
	var carriersFile CarriersFile
	defaultCarriers = internal.Must(carriersFile, yaml.Unmarshal(defaultCarriersYAML, &carriersFile)).Carriers
}
