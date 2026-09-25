package config

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// tree decodes a JSON literal the way Load does, so the test cases read as the
// mklv.config.json a project would write.
func tree(t *testing.T, s string) any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("bad test JSON %s: %v", s, err)
	}
	return Normalize(v)
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		override string
		want     string
	}{
		// Maps merge deeply.
		{"new key is added", `{"a":1}`, `{"b":2}`, `{"a":1,"b":2}`},
		{"empty override keeps base", `{"a":1,"b":[1]}`, `{}`, `{"a":1,"b":[1]}`},
		{
			"nested maps merge",
			`{"a":{"x":1,"y":2}}`,
			`{"a":{"y":3,"z":4}}`,
			`{"a":{"x":1,"y":3,"z":4}}`,
		},
		{
			"deeply nested maps merge",
			`{"a":{"b":{"c":{"d":1,"e":2}}}}`,
			`{"a":{"b":{"c":{"e":3}}}}`,
			`{"a":{"b":{"c":{"d":1,"e":3}}}}`,
		},
		{"map added under new key", `{}`, `{"a":{"b":1}}`, `{"a":{"b":1}}`},

		// Lists are appended.
		{"list appends", `{"l":[1,2]}`, `{"l":[3]}`, `{"l":[1,2,3]}`},
		{"list appends duplicates", `{"l":["a"]}`, `{"l":["a"]}`, `{"l":["a","a"]}`},
		{"empty list appends nothing", `{"l":[1]}`, `{"l":[]}`, `{"l":[1]}`},
		{"nested list appends", `{"a":{"l":[1]}}`, `{"a":{"l":[2]}}`, `{"a":{"l":[1,2]}}`},
		{
			"list of objects appends",
			`{"l":[{"x":1}]}`,
			`{"l":[{"y":2}]}`,
			`{"l":[{"x":1},{"y":2}]}`,
		},
		{"top-level lists append", `[1]`, `[2]`, `[1,2]`},

		// {"replace": [...]} replaces.
		{"replace replaces list", `{"l":[1,2]}`, `{"l":{"replace":[3]}}`, `{"l":[3]}`},
		{"replace with empty list clears", `{"l":[1,2]}`, `{"l":{"replace":[]}}`, `{"l":[]}`},
		{"replace of missing key", `{}`, `{"l":{"replace":[1]}}`, `{"l":[1]}`},
		{"replace replaces scalar", `{"l":"x"}`, `{"l":{"replace":["y"]}}`, `{"l":["y"]}`},
		{
			"nested replace",
			`{"a":{"b":{"l":[1]}}}`,
			`{"a":{"b":{"l":{"replace":[2]}}}}`,
			`{"a":{"b":{"l":[2]}}}`,
		},
		{"top-level replace", `[1]`, `{"replace":[2]}`, `[2]`},
		{
			"replace inside new value is unwrapped",
			`{}`,
			`{"a":{"l":{"replace":[1]}}}`,
			`{"a":{"l":[1]}}`,
		},
		{
			"replace inside appended element is unwrapped",
			`{"l":[]}`,
			`{"l":[{"m":{"replace":[1]}}]}`,
			`{"l":[{"m":[1]}]}`,
		},
		{
			"replace list elements are cleaned",
			`{"l":[1]}`,
			`{"l":{"replace":[{"a":null,"b":1}]}}`,
			`{"l":[{"b":1}]}`,
		},
		{
			"replace with other keys is a plain map",
			`{"a":{"x":1}}`,
			`{"a":{"replace":[1],"x":2}}`,
			`{"a":{"replace":[1],"x":2}}`,
		},
		{
			"replace of non-list is a plain map",
			`{"a":{"x":1}}`,
			`{"a":{"replace":"y"}}`,
			`{"a":{"replace":"y","x":1}}`,
		},

		// null removes the key.
		{"null removes key", `{"a":1,"b":2}`, `{"a":null}`, `{"b":2}`},
		{"null removes nested key", `{"a":{"x":1,"y":2}}`, `{"a":{"x":null}}`, `{"a":{"y":2}}`},
		{"null removes map", `{"a":{"x":1},"b":1}`, `{"a":null}`, `{"b":1}`},
		{"null removes list", `{"a":[1],"b":1}`, `{"a":null}`, `{"b":1}`},
		{"null of missing key is a no-op", `{"a":1}`, `{"b":null}`, `{"a":1}`},
		{"null inside new value is dropped", `{}`, `{"a":{"x":null,"y":1}}`, `{"a":{"y":1}}`},
		{"null list element is kept", `{"l":[1]}`, `{"l":[null]}`, `{"l":[1,null]}`},
		{"top-level null removes everything", `{"a":1}`, `null`, `null`},

		// Scalars override.
		{"scalar overrides scalar", `{"a":1}`, `{"a":2}`, `{"a":2}`},
		{"scalar changes type", `{"a":1}`, `{"a":"x"}`, `{"a":"x"}`},
		{"false overrides true", `{"a":true}`, `{"a":false}`, `{"a":false}`},
		{"float overrides int", `{"a":1}`, `{"a":1.5}`, `{"a":1.5}`},
		{
			"scalar overrides map",
			`{"MD024":{"siblings_only":true}}`,
			`{"MD024":false}`,
			`{"MD024":false}`,
		},
		{
			"map overrides scalar",
			`{"MD013":false}`,
			`{"MD013":{"line_length":120}}`,
			`{"MD013":{"line_length":120}}`,
		},
		{"scalar overrides list", `{"a":[1]}`, `{"a":"x"}`, `{"a":"x"}`},
		{"list overrides scalar", `{"a":"x"}`, `{"a":[1]}`, `{"a":[1]}`},
		{"map overrides null base", `{"a":null}`, `{"a":{"x":1}}`, `{"a":{"x":1}}`},
		{"null base value is dropped", `{"a":null,"b":1}`, `{}`, `{"b":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tree(t, tt.base)
			override := tree(t, tt.override)
			baseCopy := tree(t, tt.base)
			overrideCopy := tree(t, tt.override)
			got, err := Merge(base, override)
			if err != nil {
				t.Fatalf("Merge: %v", err)
			}
			if want := tree(t, tt.want); !reflect.DeepEqual(got, want) {
				t.Errorf("Merge(%s, %s)\n got  %#v\n want %#v", tt.base, tt.override, got, want)
			}
			if !reflect.DeepEqual(base, baseCopy) || !reflect.DeepEqual(override, overrideCopy) {
				t.Errorf("Merge modified its inputs")
			}
		})
	}
}

func TestMergeDoesNotAlias(t *testing.T) {
	base := tree(t, `{"a":{"l":[1]},"b":[{"x":1}]}`)
	got, err := Merge(base, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	got.(map[string]any)["a"].(map[string]any)["l"].([]any)[0] = int64(9)
	got.(map[string]any)["b"].([]any)[0].(map[string]any)["x"] = int64(9)
	if want := tree(t, `{"a":{"l":[1]},"b":[{"x":1}]}`); !reflect.DeepEqual(base, want) {
		t.Errorf("result shares memory with base: base is now %v", base)
	}
}

func TestMergeConflicts(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		override string
		want     ConflictError
		msg      string
	}{
		{
			"object onto list",
			`{"linters":{"enable":["a"]}}`,
			`{"linters":{"enable":{"b":true}}}`,
			ConflictError{Path: "linters.enable", Base: "list", Override: "object"},
			`linters.enable: cannot merge an object onto a list; write {"replace": [...]} to replace the list`,
		},
		{
			"list onto object",
			`{"a":{"b":{"c":{"x":1}}}}`,
			`{"a":{"b":{"c":[1]}}}`,
			ConflictError{Path: "a.b.c", Base: "object", Override: "list"},
			`a.b.c: cannot merge a list onto an object`,
		},
		{
			"replace onto object",
			`{"a":{"x":1}}`,
			`{"a":{"replace":[1]}}`,
			ConflictError{Path: "a", Base: "object", Override: "list"},
			`a: cannot merge a list onto an object`,
		},
		{
			"top-level list onto object",
			`{"a":1}`,
			`[1]`,
			ConflictError{Path: "(root)", Base: "object", Override: "list"},
			`(root): cannot merge a list onto an object`,
		},
		{
			"top-level object onto list",
			`[1]`,
			`{"a":1}`,
			ConflictError{Path: "(root)", Base: "list", Override: "object"},
			`(root): cannot merge an object onto a list; write {"replace": [...]} to replace the list`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Merge(tree(t, tt.base), tree(t, tt.override))
			var ce *ConflictError
			if !errors.As(err, &ce) {
				t.Fatalf("Merge = %v, %v; want a *ConflictError", got, err)
			}
			if *ce != tt.want {
				t.Errorf("conflict = %+v, want %+v", *ce, tt.want)
			}
			if err.Error() != tt.msg {
				t.Errorf("message = %q\nwant      %q", err.Error(), tt.msg)
			}
		})
	}
}

// TestMergeConflictPathsAreIndependent guards against sibling keys sharing the
// path slice's backing array, which would report the wrong key.
func TestMergeConflictPathsAreIndependent(t *testing.T) {
	base := tree(t, `{"a":{"b":{"c":{"d":1},"e":{"f":1}}}}`)
	_, err := Merge(base, tree(t, `{"a":{"b":{"e":[1]}}}`))
	var ce *ConflictError
	if !errors.As(err, &ce) || ce.Path != "a.b.e" {
		t.Fatalf("err = %v, want a conflict at a.b.e", err)
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{"json integer", json.Number("30"), int64(30)},
		{"json float", json.Number("0.5"), 0.5},
		{"json exponent", json.Number("1e3"), 1000.0},
		{"yaml int", 30, int64(30)},
		{"int64 kept", int64(7), int64(7)},
		{"float kept", 1.5, 1.5},
		{"string kept", "x", "x"},
		{"bool kept", true, true},
		{"nil kept", nil, nil},
		{
			"nested map",
			map[string]any{"a": map[string]any{"b": 1}},
			map[string]any{"a": map[string]any{"b": int64(1)}},
		},
		{"list", []any{1, "x"}, []any{int64(1), "x"}},
		{
			"toml array of tables",
			[]map[string]any{{"a": 1}},
			[]any{map[string]any{"a": int64(1)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Normalize(%#v) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
