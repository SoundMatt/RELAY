// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// This is a deliberately small JSON Schema validator covering only the subset
// of draft 2020-12 used by the schemas in spec/schemas/. Keeping it dependency-
// free and auditable matters more for a functional-safety tool than supporting
// the full vocabulary. Supported keywords:
//
//	type (string or []string), properties, required, additionalProperties:false,
//	enum, const, minimum, maximum, items (single schema), minItems, maxItems,
//	contains (single schema).
//
// Unsupported keywords are ignored (they do not cause false failures).

// validateSchema validates doc against the JSON Schema in schemaJSON.
// It returns a sorted list of human-readable violation messages; an empty
// slice means the document conforms.
//
//fusa:req REQ-RELAY-058
func validateSchema(schemaJSON []byte, doc interface{}) []string {
	var schema map[string]interface{}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return []string{fmt.Sprintf("schema is not valid JSON: %v", err)}
	}
	var violations []string
	validateValue(schema, doc, "", &violations)
	sort.Strings(violations)
	return violations
}

func validateValue(schema map[string]interface{}, v interface{}, path string, out *[]string) {
	loc := path
	if loc == "" {
		loc = "(root)"
	}

	if !typeMatches(schema["type"], v) {
		*out = append(*out, fmt.Sprintf("%s: expected type %v, got %s", loc, schema["type"], jsonTypeOf(v)))
		return
	}

	if c, ok := schema["const"]; ok && !jsonEqual(c, v) {
		*out = append(*out, fmt.Sprintf("%s: must equal %v, got %v", loc, c, v))
	}

	if enum, ok := schema["enum"].([]interface{}); ok {
		matched := false
		for _, e := range enum {
			if jsonEqual(e, v) {
				matched = true
				break
			}
		}
		if !matched {
			*out = append(*out, fmt.Sprintf("%s: value %v not in enum %v", loc, v, enum))
		}
	}

	switch val := v.(type) {
	case float64:
		if min, ok := numberOf(schema["minimum"]); ok && val < min {
			*out = append(*out, fmt.Sprintf("%s: %v < minimum %v", loc, val, min))
		}
		if max, ok := numberOf(schema["maximum"]); ok && val > max {
			*out = append(*out, fmt.Sprintf("%s: %v > maximum %v", loc, val, max))
		}
	case map[string]interface{}:
		validateObject(schema, val, path, out)
	case []interface{}:
		validateArray(schema, val, path, out)
	}
}

func validateObject(schema map[string]interface{}, obj map[string]interface{}, path string, out *[]string) {
	props, _ := schema["properties"].(map[string]interface{})

	if req, ok := schema["required"].([]interface{}); ok {
		for _, r := range req {
			name, _ := r.(string)
			if _, present := obj[name]; !present {
				*out = append(*out, fmt.Sprintf("%s: missing required property %q", pathName(path), name))
			}
		}
	}

	if ap, ok := schema["additionalProperties"]; ok {
		if allowed, isBool := ap.(bool); isBool && !allowed {
			for key := range obj {
				if _, declared := props[key]; !declared {
					*out = append(*out, fmt.Sprintf("%s: additional property %q is not allowed", pathName(path), key))
				}
			}
		}
	}

	for key, sub := range props {
		subSchema, ok := sub.(map[string]interface{})
		if !ok {
			continue
		}
		if child, present := obj[key]; present {
			validateValue(subSchema, child, joinPath(path, key), out)
		}
	}
}

func validateArray(schema map[string]interface{}, arr []interface{}, path string, out *[]string) {
	if min, ok := numberOf(schema["minItems"]); ok && float64(len(arr)) < min {
		*out = append(*out, fmt.Sprintf("%s: array length %d < minItems %v", pathName(path), len(arr), min))
	}
	if max, ok := numberOf(schema["maxItems"]); ok && float64(len(arr)) > max {
		*out = append(*out, fmt.Sprintf("%s: array length %d > maxItems %v", pathName(path), len(arr), max))
	}
	if items, ok := schema["items"].(map[string]interface{}); ok {
		for i, el := range arr {
			validateValue(items, el, fmt.Sprintf("%s[%d]", path, i), out)
		}
	}
	if contains, ok := schema["contains"].(map[string]interface{}); ok {
		found := false
		for _, el := range arr {
			var ignore []string
			validateValue(contains, el, "", &ignore)
			if len(ignore) == 0 {
				found = true
				break
			}
		}
		if !found {
			*out = append(*out, fmt.Sprintf("%s: array does not contain a value matching the 'contains' schema", pathName(path)))
		}
	}
}

// typeMatches reports whether v satisfies the schema "type" keyword.
// A nil/absent type matches anything.
func typeMatches(t interface{}, v interface{}) bool {
	switch tt := t.(type) {
	case nil:
		return true
	case string:
		return jsonTypeIs(tt, v)
	case []interface{}:
		for _, alt := range tt {
			if s, ok := alt.(string); ok && jsonTypeIs(s, v) {
				return true
			}
		}
		return false
	}
	return true
}

func jsonTypeIs(typ string, v interface{}) bool {
	switch typ {
	case "object":
		_, ok := v.(map[string]interface{})
		return ok
	case "array":
		_, ok := v.([]interface{})
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "null":
		return v == nil
	case "number":
		_, ok := v.(float64)
		return ok
	case "integer":
		f, ok := v.(float64)
		return ok && f == math.Trunc(f)
	}
	return false
}

func jsonTypeOf(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		if x == math.Trunc(x) {
			return "integer"
		}
		return "number"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	}
	return "unknown"
}

func jsonEqual(a, b interface{}) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func numberOf(v interface{}) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func pathName(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

// schemaTitle extracts the human-readable title from a schema, for messages.
func schemaTitle(schemaJSON []byte) string {
	var s struct {
		Title string `json:"title"`
	}
	_ = json.Unmarshal(schemaJSON, &s)
	if s.Title == "" {
		return "document"
	}
	return strings.TrimSpace(s.Title)
}
