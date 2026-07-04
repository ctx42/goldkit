package goldkit

import (
	"gopkg.in/yaml.v3"
)

// base is a collection of YAML fields common among a few golden file types.
type base struct {
	// Meta is a free-form key-value store.
	Meta map[string]any `yaml:"meta,omitempty"`

	// Golden file content type: text, json, multipart, ...
	// If not specified, it will be set to 'text' by default.
	BodyType string `yaml:"bodyType"`

	// Raw YAML node representing the golden file body. The node is its zero
	// value (RawBody.Value is empty) when the "body" field in YAML is not set.
	RawBody yaml.Node `yaml:"body"`
}

// M returns Meta field value.
func (b *base) M() Meta { return b.Meta }
