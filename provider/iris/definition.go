package iris

import (
	_ "embed"
	"encoding/json"
	"strings"
)

type operation struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}
type definition struct {
	Name           string    `json:"name"`
	Service        string    `json:"service"`
	IDField        string    `json:"id_field"`
	Create         operation `json:"create"`
	Read           operation `json:"read"`
	Update         operation `json:"update"`
	Delete         operation `json:"delete"`
	Fields         []field   `json:"fields"`
	Revision       bool      `json:"revision"`
	DeleteRevision bool      `json:"delete_revision"`
}
type field struct {
	Name      string         `json:"name"`
	Wire      string         `json:"wire"`
	JSON      bool           `json:"json"`
	Create    bool           `json:"create"`
	Update    bool           `json:"update"`
	Read      bool           `json:"read"`
	Path      bool           `json:"path"`
	Shape     map[string]any `json:"shape"`
	ReadShape map[string]any `json:"read_shape"`
}

//go:embed definitions_gen.json
var definitionJSON []byte
var definitions = loadDefinitions()

func loadDefinitions() []definition {
	var defs []definition
	if err := json.Unmarshal(definitionJSON, &defs); err != nil {
		panic(err)
	}
	return defs
}
func (d definition) parentFields() []field {
	var out []field
	for _, f := range d.Fields {
		if f.Path && f.Wire != d.IDField {
			out = append(out, f)
		}
	}
	return out
}
func (d definition) pathField(wire string) string {
	if wire == "memoryTypeName" {
		wire = "name"
	}
	for _, f := range d.Fields {
		if f.Wire == wire {
			return f.Name
		}
	}
	return strings.ToLower(wire)
}
