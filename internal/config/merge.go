// Package config loads a project's mklv.config.json and merges its overrides
// onto the base configs (ADR 0005).
//
// Config values are generic trees, as decoded from JSON, YAML or TOML: objects
// are map[string]any, lists are []any, and everything else is a scalar. Merge
// and Normalize work on those trees.
package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// replaceKey is the only key of the object form that replaces a base list,
// e.g. {"replace": ["a", "b"]}.
const replaceKey = "replace"

// ConflictError reports an override that cannot be merged onto its base value:
// an object where the base has a list, or a list where the base has an object.
// Either is almost certainly a mistake in the override, so it is an error rather
// than a silent replacement.
type ConflictError struct {
	// Path is the dotted path of the conflicting key, e.g. "linters.enable".
	Path string
	// Base and Override name the kinds found, "object" or "list".
	Base, Override string
}

func (e *ConflictError) Error() string {
	hint := ""
	if e.Base == "list" {
		hint = `; write {"replace": [...]} to replace the list`
	}
	return fmt.Sprintf("%s: cannot merge %s onto %s%s", e.Path, article(e.Override),
		article(e.Base), hint)
}

func article(kind string) string {
	if kind == "object" {
		return "an object"
	}
	return "a list"
}

// Merge returns override merged onto base, following ADR 0005:
//
//   - Objects merge deeply, key by key.
//   - A list is appended to the base list.
//   - An object whose only key is "replace", holding a list, replaces the base
//     list (or scalar) with that list.
//   - A null value removes its key. A null override as a whole returns nil.
//   - Any other value overrides the base, including a scalar replacing an
//     object or list and an object or list replacing a scalar.
//
// An object merged onto a list, a list merged onto an object, or a replace form
// merged onto an object returns a *ConflictError naming the key.
//
// Neither input is modified. Values that come only from override are
// normalized with the same rules, so nulls and replace forms never reach the
// result.
func Merge(base, override any) (any, error) {
	return merge(nil, base, override)
}

func merge(path []string, base, override any) (any, error) {
	if override == nil {
		return nil, nil
	}
	if list, ok := replaceList(override); ok {
		if _, isMap := base.(map[string]any); isMap {
			return nil, &ConflictError{Path: join(path), Base: "object", Override: "list"}
		}
		return clean(list), nil
	}
	switch o := override.(type) {
	case map[string]any:
		switch b := base.(type) {
		case map[string]any:
			return mergeMaps(path, b, o)
		case []any:
			return nil, &ConflictError{Path: join(path), Base: "list", Override: "object"}
		}
	case []any:
		switch b := base.(type) {
		case []any:
			out := make([]any, 0, len(b)+len(o))
			out = append(out, cleanList(b)...)
			return append(out, cleanList(o)...), nil
		case map[string]any:
			return nil, &ConflictError{Path: join(path), Base: "object", Override: "list"}
		}
	}
	return clean(override), nil
}

func mergeMaps(path []string, base, override map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		if v != nil {
			out[k] = clean(v)
		}
	}
	for k, v := range override {
		if v == nil {
			delete(out, k)
			continue
		}
		b, ok := out[k]
		if !ok {
			out[k] = clean(v)
			continue
		}
		m, err := merge(append(path[:len(path):len(path)], k), b, v)
		if err != nil {
			return nil, err
		}
		out[k] = m
	}
	return out, nil
}

// replaceList reports whether v is the replace form, {"replace": [...]}, and
// returns its list.
func replaceList(v any) ([]any, bool) {
	m, ok := v.(map[string]any)
	if !ok || len(m) != 1 {
		return nil, false
	}
	list, ok := m[replaceKey].([]any)
	return list, ok
}

// clean deep-copies a value, dropping null object values and unwrapping replace
// forms, so a value with no base to merge onto follows the same rules as one
// merged onto a base. Null list elements are kept: null only removes keys.
func clean(v any) any {
	if list, ok := replaceList(v); ok {
		return clean(list)
	}
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			if e != nil {
				out[k] = clean(e)
			}
		}
		return out
	case []any:
		return cleanList(t)
	}
	return v
}

// cleanList returns a copy of list with clean applied to each element.
func cleanList(list []any) []any {
	out := make([]any, len(list))
	for i, e := range list {
		out[i] = clean(e)
	}
	return out
}

func join(path []string) string {
	if len(path) == 0 {
		return "(root)"
	}
	return strings.Join(path, ".")
}

// Normalize converts a tree decoded by encoding/json (with UseNumber), YAML or
// TOML into the canonical form Merge and the generators expect: integral
// numbers become int64, other numbers float64, and typed maps and slices
// become map[string]any and []any. Normalizing the base and the override the
// same way is what makes a JSON 30 and a YAML 30 the same value.
func Normalize(v any) any {
	switch t := v.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		f, _ := t.Float64()
		return f
	case int:
		return int64(t)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = Normalize(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = Normalize(e)
		}
		return out
	case []map[string]any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = Normalize(e)
		}
		return out
	}
	return v
}
