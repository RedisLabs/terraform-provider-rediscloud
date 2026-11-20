// irisgen adapts the service specifications to protocol-v5-compatible schemas.
package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type object = map[string]any

type operation struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}
type resourceSpec struct {
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
	Name      string `json:"name"`
	Wire      string `json:"wire"`
	JSON      bool   `json:"json"`
	Create    bool   `json:"create"`
	Update    bool   `json:"update"`
	Read      bool   `json:"read"`
	Path      bool   `json:"path"`
	Shape     object `json:"shape,omitempty"`
	ReadShape object `json:"read_shape,omitempty"`
}

var specs = []resourceSpec{
	{Name: "iris_agent_memory", Service: "agent-memory", IDField: "storeId", Create: operation{"/iris/v1/agent-memory/stores", "POST"}, Read: operation{"/iris/v1/agent-memory/stores/{storeId}", "GET"}, Update: operation{"/iris/v1/agent-memory/stores/{storeId}", "PATCH"}, Delete: operation{"/iris/v1/agent-memory/stores/{storeId}", "DELETE"}},
	{Name: "iris_agent_memory_extractor", Service: "agent-memory", IDField: "extractorId", Create: operation{"/iris/v1/agent-memory/stores/{storeId}/extractors", "POST"}, Read: operation{"/iris/v1/agent-memory/stores/{storeId}/extractors/{extractorId}", "GET"}, Update: operation{"/iris/v1/agent-memory/stores/{storeId}/extractors/{extractorId}", "PATCH"}, Delete: operation{"/iris/v1/agent-memory/stores/{storeId}/extractors/{extractorId}", "DELETE"}},
	{Name: "iris_agent_memory_type", Service: "agent-memory", IDField: "name", Create: operation{"/iris/v1/agent-memory/stores/{storeId}/memory-types", "POST"}, Read: operation{"/iris/v1/agent-memory/stores/{storeId}/memory-types/{memoryTypeName}", "GET"}, Update: operation{"/iris/v1/agent-memory/stores/{storeId}/memory-types/{memoryTypeName}", "PATCH"}, Delete: operation{"/iris/v1/agent-memory/stores/{storeId}/memory-types/{memoryTypeName}", "DELETE"}},
	{Name: "iris_identity_api_key", Service: "identity", IDField: "keyId", Create: operation{"/v1/api-keys", "POST"}, Read: operation{"/v1/api-keys/{keyId}", "GET"}, Update: operation{"/v1/api-keys/{keyId}", "PATCH"}, Delete: operation{"/v1/api-keys/{keyId}", "DELETE"}},
	{Name: "iris_identity_provider", Service: "identity", IDField: "name", Create: operation{"/v1/tenants/{tenant}/identity-providers", "POST"}, Read: operation{"/v1/tenants/{tenant}/identity-providers/{name}", "GET"}, Update: operation{"/v1/tenants/{tenant}/identity-providers/{name}", "PUT"}, Delete: operation{"/v1/tenants/{tenant}/identity-providers/{name}", "DELETE"}},
	{Name: "iris_identity_subject_grant_set", Service: "identity", IDField: "id", Create: operation{"/v1/subject-grants", "POST"}, Read: operation{"/v1/subject-grants/{id}", "GET"}, Delete: operation{"/v1/subject-grants/{id}", "DELETE"}},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func readJSON(p string) (object, error) {
	b, e := os.ReadFile(p)
	if e != nil {
		return nil, e
	}
	var v object
	e = json.Unmarshal(b, &v)
	return v, e
}
func writeJSON(p string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(p, append(b, '\n'), 0644)
}
func obj(v any) object {
	m, _ := v.(map[string]any)
	if m == nil {
		return object{}
	}
	return m
}
func str(v any) string { s, _ := v.(string); return s }
func sorted(m object) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func contains(v any, s string) bool {
	for _, x := range arr(v) {
		if x == s {
			return true
		}
	}
	return false
}
func arr(v any) []any { a, _ := v.([]any); return a }
func snake(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if unicode.IsUpper(c) && i > 0 && (unicode.IsLower(r[i-1]) || (i+1 < len(r) && unicode.IsLower(r[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}
func resolve(doc object, s object, depth int) (object, error) {
	if depth > 30 {
		return nil, fmt.Errorf("recursive schema is unsupported")
	}
	if ref := str(s["$ref"]); ref != "" {
		prefix := "#/components/schemas/"
		if !strings.HasPrefix(ref, prefix) {
			return nil, fmt.Errorf("unsupported ref %q", ref)
		}
		target, ok := obj(obj(doc["components"])["schemas"])[strings.TrimPrefix(ref, prefix)]
		if !ok {
			return nil, fmt.Errorf("missing ref %q", ref)
		}
		return resolve(doc, obj(target), depth+1)
	}
	result := object{}
	for k, v := range s {
		result[k] = v
	}
	if properties, ok := s["properties"]; ok {
		p := object{}
		for k, v := range obj(properties) {
			x, e := resolve(doc, obj(v), depth+1)
			if e != nil {
				return nil, e
			}
			p[k] = x
		}
		result["properties"] = p
	}
	for _, k := range []string{"items", "additionalProperties"} {
		if v, ok := s[k].(map[string]any); ok {
			x, e := resolve(doc, v, depth+1)
			if e != nil {
				return nil, e
			}
			result[k] = x
		}
	}
	return result, nil
}
func op(doc object, o operation) object {
	return obj(obj(obj(doc["paths"])[o.Path])[strings.ToLower(o.Method)])
}
func request(doc object, o operation) (object, error) {
	return resolve(doc, obj(obj(obj(obj(op(doc, o)["requestBody"])["content"])["application/json"])["schema"]), 0)
}
func response(doc object, o operation) (object, error) {
	for _, code := range sorted(obj(op(doc, o)["responses"])) {
		if strings.HasPrefix(code, "2") {
			return resolve(doc, obj(obj(obj(obj(obj(op(doc, o)["responses"])[code])["content"])["application/json"])["schema"]), 0)
		}
	}
	return object{}, nil
}
func complex(s object) bool {
	t := str(s["type"])
	if t == "array" {
		item := obj(s["items"])
		return complex(item) || str(item["type"]) == "array"
	}
	return t == "object" || t == ""
}

var unicodeEscape = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

func pattern(p string) (string, error) {
	p = strings.ReplaceAll(p, `\\`, `\`)
	// Tenant path constraints also need a semantic check for dot segments and whitespace.
	p = strings.ReplaceAll(p, `(?!\.{1,2}$)`, "")
	p = unicodeEscape.ReplaceAllString(p, `\x{$1}`)
	_, e := regexp.Compile(p)
	return p, e
}
func safeSchema(s object) (object, error) {
	b, _ := json.Marshal(s)
	var out object
	_ = json.Unmarshal(b, &out)
	if complex(s) {
		out = object{"type": "string", "description": "JSON-encoded service configuration. Use jsonencode(...) in Terraform. " + str(s["description"])}
		if strings.Contains(fmt.Sprint(s), "password") {
			out["format"] = "password"
		}
		return out, nil
	}
	if p := str(out["pattern"]); p != "" {
		fixed, e := pattern(p)
		if e != nil {
			return nil, fmt.Errorf("unsupported regex %q: %w", p, e)
		}
		out["pattern"] = fixed
	}
	if str(out["type"]) == "array" {
		x, e := safeSchema(obj(s["items"]))
		if e != nil {
			return nil, e
		}
		out["items"] = x
	}
	return out, nil
}
func run() error {
	base := "scripts/iris-codegen"
	if len(os.Args) > 1 && os.Args[1] == "adapt" {
		return adapt(base)
	}
	if e := os.MkdirAll(base+"/prepared", 0755); e != nil {
		return e
	}
	docs := map[string]object{}
	configs := map[string]object{}
	for _, service := range []string{"agent-memory", "identity"} {
		d, e := readJSON(base + "/inputs/" + service + ".openapi.json")
		if e != nil {
			return e
		}
		docs[service] = d
		configs[service] = object{"provider": object{"name": "rediscloud"}, "resources": object{}}
	}
	for i := range specs {
		spec := &specs[i]
		doc := docs[spec.Service]
		create, e := request(doc, spec.Create)
		if e != nil {
			return e
		}
		update, e := request(doc, spec.Update)
		if e != nil {
			return e
		}
		read, e := response(doc, spec.Read)
		if e != nil {
			return e
		}
		created, e := response(doc, spec.Create)
		if e != nil {
			return e
		}
		fields := object{}
		for _, s := range []object{read, created, create} {
			for k, v := range obj(s["properties"]) {
				fields[k] = v
			}
		}
		paths := map[string]bool{}
		for _, o := range []operation{spec.Create, spec.Read} {
			for _, raw := range arr(op(doc, o)["parameters"]) {
				p := obj(raw)
				if p["in"] == "path" {
					k := str(p["name"])
					if k == "memoryTypeName" {
						k = "name"
					}
					paths[k] = true
					if fields[k] == nil {
						v, e := resolve(doc, obj(p["schema"]), 0)
						if e != nil {
							return e
						}
						fields[k] = v
					}
				}
			}
		}
		delete(fields, "expectedRevision")
		for _, wire := range sorted(fields) {
			shape := obj(fields[wire])
			name := snake(wire)
			isJSON := complex(shape)
			if isJSON {
				name += "_json"
			}
			_, c := obj(create["properties"])[wire]
			if wire == "revision" {
				c = false
			}
			_, u := obj(update["properties"])[wire]
			_, r := obj(read["properties"])[wire]
			spec.Fields = append(spec.Fields, field{Name: name, Wire: wire, JSON: isJSON, Create: c, Update: u, Read: r, Path: paths[wire], Shape: shape, ReadShape: obj(obj(read["properties"])[wire])})
		}
		_, spec.Revision = obj(update["properties"])["expectedRevision"]
		for _, raw := range arr(op(doc, spec.Delete)["parameters"]) {
			if obj(raw)["name"] == "expectedRevision" {
				spec.DeleteRevision = true
			}
		}
		config := object{"create": spec.Create, "read": spec.Read, "delete": spec.Delete, "schema": object{"ignores": []string{"projectId", "expectedRevision"}}}
		if spec.Update.Path != "" {
			config["update"] = spec.Update
		}
		obj(configs[spec.Service]["resources"])[spec.Name] = config
	}
	// Rewrite only operation-level bodies/parameters; original inputs remain untouched.
	for service, doc := range docs {
		for _, rawPath := range obj(doc["paths"]) {
			for method, rawOp := range obj(rawPath) {
				if !contains([]any{"get", "post", "patch", "put", "delete"}, method) {
					continue
				}
				o := obj(rawOp)
				for _, param := range arr(o["parameters"]) {
					p := obj(param)
					shape, e := resolve(doc, obj(p["schema"]), 0)
					if e != nil {
						return e
					}
					safe, e := safeSchema(shape)
					if e != nil {
						return e
					}
					p["schema"] = safe
				}
				containers := []object{obj(o["requestBody"])}
				for _, r := range obj(o["responses"]) {
					containers = append(containers, obj(r))
				}
				for _, container := range containers {
					media := obj(obj(container["content"])["application/json"])
					if media["schema"] == nil {
						continue
					}
					shape, e := resolve(doc, obj(media["schema"]), 0)
					if e != nil {
						return e
					}
					props := object{}
					for k, v := range obj(shape["properties"]) {
						safe, e := safeSchema(obj(v))
						if e != nil {
							return e
						}
						props[k] = safe
					}
					shape["properties"] = props
					media["schema"] = shape
				}
			}
		}
		if e := writeJSON(base+"/prepared/"+service+".openapi.json", doc); e != nil {
			return e
		}
		if e := writeJSON(base+"/prepared/"+service+".config.json", configs[service]); e != nil {
			return e
		}
	}
	return writeJSON(base+"/prepared/definitions.json", specs)
}

func modifier(kind, definition string) object {
	return object{"custom": object{"imports": []object{{"path": "github.com/hashicorp/terraform-plugin-framework/resource/schema/" + kind + "planmodifier"}}, "schema_definition": kind + "planmodifier." + definition + "()"}}
}
func adapt(base string) error {
	b, e := os.ReadFile(base + "/prepared/definitions.json")
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, &specs); e != nil {
		return e
	}
	for _, service := range []string{"agent-memory", "identity"} {
		doc, e := readJSON(base + "/prepared/" + service + ".spec.json")
		if e != nil {
			return e
		}
		resources := arr(doc["resources"])
		expected := 0
		for _, s := range specs {
			if s.Service == service {
				expected++
			}
		}
		if len(resources) != expected {
			return fmt.Errorf("%s: generated %d resources, expected %d; inspect generator warnings", service, len(resources), expected)
		}
		for _, raw := range resources {
			r := obj(raw)
			name := str(r["name"])
			var spec *resourceSpec
			for i := range specs {
				if specs[i].Name == name {
					spec = &specs[i]
					break
				}
			}
			if spec == nil {
				return fmt.Errorf("unexpected resource %q", name)
			}
			fields := map[string]field{}
			for _, f := range spec.Fields {
				fields[snake(f.Wire)] = f
			}
			var attrs []any
			for _, rawAttr := range arr(obj(r["schema"])["attributes"]) {
				a := obj(rawAttr)
				f, ok := fields[str(a["name"])]
				if !ok {
					continue
				}
				a["name"] = f.Name
				var kind string
				var value object
				for _, k := range []string{"string", "int64", "bool", "list", "set", "float64"} {
					if v, ok := a[k]; ok {
						kind = k
						value = obj(v)
						break
					}
				}
				if kind == "" {
					return fmt.Errorf("%s.%s: unadapted complex attribute", name, f.Name)
				}
				mode := "computed"
				if f.Create || f.Path {
					mode = "computed_optional"
					if f.Path || contains(mustRequestRequired(service, *spec), f.Wire) {
						mode = "required"
					}
				}
				if f.Wire == spec.IDField && !f.Create && !strings.Contains(spec.Create.Path, "{"+f.Wire+"}") {
					mode = "computed"
				}
				value["computed_optional_required"] = mode
				if f.JSON {
					value["description"] = "JSON-encoded service configuration. Use jsonencode(...). " + str(f.Shape["description"])
					delete(value, "validators")
				}
				if mode != "computed" {
					pmKind := kind
					if kind == "int64" {
						pmKind = "int64"
					}
					if f.Path || !f.Update {
						replace := "RequiresReplace"
						if mode == "computed_optional" {
							replace = "RequiresReplaceIfConfigured"
						}
						value["plan_modifiers"] = []any{modifier(pmKind, replace)}
					}
				}
				if mode == "computed" && f.Wire == spec.IDField {
					value["plan_modifiers"] = []any{modifier("string", "UseStateForUnknown")}
				}
				// Read-only validators add no value and may describe a different response shape.
				if mode == "computed" {
					delete(value, "validators")
				}
				attrs = append(attrs, a)
			}
			if spec.Name == "iris_agent_memory" {
				attrs = append(attrs, object{"name": "flush_on_delete", "bool": object{"computed_optional_required": "computed_optional", "default": object{"static": false}, "description": "Delete short-term sessions on destroy. Defaults to false; long-term memories are retained."}})
			}
			if spec.IDField != "id" {
				attrs = append(attrs, object{"name": "id", "string": object{"computed_optional_required": "computed", "description": "Resource identifier used for import.", "plan_modifiers": []any{modifier("string", "UseStateForUnknown")}}})
			}
			obj(r["schema"])["attributes"] = attrs
		}
		if e = writeJSON(base+"/prepared/"+service+".adapted.json", doc); e != nil {
			return e
		}
	}
	if e = os.MkdirAll("provider/iris", 0755); e != nil {
		return e
	}
	if e = writeJSON("provider/iris/definitions_gen.json", specs); e != nil {
		return e
	}
	var out strings.Builder
	out.WriteString("// Code generated by internal/irisgen. DO NOT EDIT.\npackage iris\nimport (\n\"context\"\n\"github.com/hashicorp/terraform-plugin-framework/resource/schema\"\n")
	for _, s := range specs {
		fmt.Fprintf(&out, "%s %s\n", s.Name, strconv.Quote("github.com/RedisLabs/terraform-provider-rediscloud/provider/iris/generated/resource_"+s.Name))
	}
	out.WriteString(")\nvar generatedSchemas=map[string]func(context.Context)schema.Schema{\n")
	for _, s := range specs {
		var pascal strings.Builder
		for _, p := range strings.Split(s.Name, "_") {
			pascal.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
		fmt.Fprintf(&out, "%q:%s.%sResourceSchema,\n", s.Name, s.Name, pascal.String())
	}
	out.WriteString("}\n")
	formatted, e := format.Source([]byte(out.String()))
	if e != nil {
		return e
	}
	return os.WriteFile(filepath.Join("provider", "iris", "schemas_gen.go"), formatted, 0644)
}
func mustRequestRequired(service string, s resourceSpec) any {
	d, e := readJSON("scripts/iris-codegen/inputs/" + service + ".openapi.json")
	if e != nil {
		panic(e)
	}
	r, e := request(d, s.Create)
	if e != nil {
		panic(e)
	}
	return r["required"]
}
