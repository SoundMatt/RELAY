// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	relay "github.com/SoundMatt/RELAY"
)

func mustValidate(t *testing.T, schema string, jsonDoc string) []string {
	t.Helper()
	var doc interface{}
	if err := json.Unmarshal([]byte(jsonDoc), &doc); err != nil {
		t.Fatalf("test doc is not valid JSON: %v", err)
	}
	return validateSchema([]byte(schema), doc)
}

//fusa:test REQ-RELAY-058
func TestSchemaTypeMismatch(t *testing.T) {
	schema := `{"type":"object","properties":{"n":{"type":"integer"}}}`
	v := mustValidate(t, schema, `{"n":"not an int"}`)
	if len(v) == 0 {
		t.Error("expected a violation for string where integer required")
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaIntegerAcceptsWholeFloat(t *testing.T) {
	schema := `{"type":"integer"}`
	if v := validateSchema([]byte(schema), float64(42)); len(v) != 0 {
		t.Errorf("42 should be a valid integer, got %v", v)
	}
	if v := validateSchema([]byte(schema), float64(4.2)); len(v) == 0 {
		t.Error("4.2 should not be a valid integer")
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaRequired(t *testing.T) {
	schema := `{"type":"object","required":["a","b"],"properties":{"a":{"type":"string"},"b":{"type":"string"}}}`
	v := mustValidate(t, schema, `{"a":"x"}`)
	if len(v) != 1 {
		t.Errorf("expected exactly one violation for missing b, got %v", v)
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaAdditionalPropertiesFalse(t *testing.T) {
	schema := `{"type":"object","additionalProperties":false,"properties":{"a":{"type":"string"}}}`
	v := mustValidate(t, schema, `{"a":"x","b":"y"}`)
	if len(v) != 1 {
		t.Errorf("expected one violation for undeclared property b, got %v", v)
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaEnumAndConst(t *testing.T) {
	enum := `{"type":"string","enum":["go","cpp","rust"]}`
	if v := validateSchema([]byte(enum), "java"); len(v) == 0 {
		t.Error("java should violate the language enum")
	}
	if v := validateSchema([]byte(enum), "go"); len(v) != 0 {
		t.Errorf("go should satisfy the enum, got %v", v)
	}
	cst := `{"const":"capabilities"}`
	if v := validateSchema([]byte(cst), "version"); len(v) == 0 {
		t.Error("'version' should violate const 'capabilities'")
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaMinMax(t *testing.T) {
	schema := `{"type":"integer","minimum":0,"maximum":63}`
	if v := validateSchema([]byte(schema), float64(64)); len(v) == 0 {
		t.Error("64 should exceed maximum 63")
	}
	if v := validateSchema([]byte(schema), float64(-1)); len(v) == 0 {
		t.Error("-1 should be below minimum 0")
	}
	if v := validateSchema([]byte(schema), float64(10)); len(v) != 0 {
		t.Errorf("10 should be in range, got %v", v)
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaArrayItemsAndContains(t *testing.T) {
	schema := `{"type":"array","items":{"type":"string"},"contains":{"const":"version"}}`
	if v := mustValidate(t, schema, `["version","status"]`); len(v) != 0 {
		t.Errorf("valid array should pass, got %v", v)
	}
	if v := mustValidate(t, schema, `["status","capabilities"]`); len(v) == 0 {
		t.Error("array without 'version' should violate contains")
	}
	if v := mustValidate(t, schema, `["version",3]`); len(v) == 0 {
		t.Error("array with non-string item should violate items")
	}
}

//fusa:test REQ-RELAY-058
func TestSchemaMinMaxItems(t *testing.T) {
	schema := `{"type":"array","minItems":16,"maxItems":16,"items":{"type":"integer"}}`
	if v := mustValidate(t, schema, `[1,2,3]`); len(v) == 0 {
		t.Error("3-element array should violate minItems 16")
	}
}

// TestGoldenVectorsConformToSchemas validates each committed golden vector's
// canonical value against its published JSON schema, closing the loop between
// spec/vectors/ and spec/schemas/.
//
//fusa:test REQ-RELAY-058
func TestGoldenVectorsConformToSchemas(t *testing.T) {
	typeToSchema := map[string]string{
		"can.Frame":      "can-frame",
		"dds.Sample":     "dds-sample",
		"lin.Frame":      "lin-frame",
		"mqtt.Message":   "mqtt-message",
		"rcp.Status":     "rcp-status",
		"someip.Message": "someip-message",
	}

	// Vectors live at the repo root; tests for cmd/relay run in this directory.
	paths, err := filepath.Glob(filepath.Join("..", "..", "spec", "vectors", "*.json"))
	if err != nil {
		t.Fatalf("glob vectors: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no golden vectors found")
	}

	for _, p := range paths {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read vector: %v", err)
			}
			var vec struct {
				Type  string          `json:"type"`
				Value json.RawMessage `json:"value"`
			}
			if err := json.Unmarshal(data, &vec); err != nil {
				t.Fatalf("unmarshal vector: %v", err)
			}
			schemaName, ok := typeToSchema[vec.Type]
			if !ok {
				t.Fatalf("no schema mapping for vector type %q", vec.Type)
			}
			schemaJSON, err := relay.Schema(schemaName)
			if err != nil {
				t.Fatalf("load schema %q: %v", schemaName, err)
			}
			var value interface{}
			if err := json.Unmarshal(vec.Value, &value); err != nil {
				t.Fatalf("unmarshal value: %v", err)
			}
			if violations := validateSchema(schemaJSON, value); len(violations) != 0 {
				t.Errorf("vector %s value does not conform to %s schema:\n%v", filepath.Base(p), schemaName, violations)
			}
		})
	}
}
