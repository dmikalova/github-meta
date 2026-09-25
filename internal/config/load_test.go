package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	p, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(p, &Project{}) {
		t.Errorf("Load of a project without %s = %+v, want the zero Project", FileName, p)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{
		"$schema": "https://example.com/schema.json",
		"name": "vex",
		"entrypoint": "cmd/web/main.go",
		"runtime": {"port": 8080, "healthCheckPath": "/health"},
		"kind": "cloudrun",
		"tools": {"golangci": {"linters": {"settings": {"gocognit": {"min-complexity": 40}}}}},
		"ignore": {"git": ["tmp/"], "docker": ["docs/"]}
	}`)
	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := &Project{
		Schema:     "https://example.com/schema.json",
		Name:       "vex",
		Entrypoint: "cmd/web/main.go",
		Runtime:    &Runtime{Port: 8080, HealthCheckPath: "/health"},
		Kind:       "cloudrun",
		Tools: map[string]any{"golangci": map[string]any{"linters": map[string]any{
			"settings": map[string]any{"gocognit": map[string]any{"min-complexity": int64(40)}},
		}}},
		Ignore: &Ignore{Git: []string{"tmp/"}, Docker: []string{"docs/"}},
	}
	if !reflect.DeepEqual(p, want) {
		t.Errorf("Load =\n %#v\nwant\n %#v", p, want)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{"invalid JSON", `{`, "mklv.config.json: unexpected EOF"},
		{"unknown field", `{"nmae": "x"}`, `unknown field "nmae"`},
		{"unknown nested field", `{"runtime": {"prot": 1}}`, `unknown field "prot"`},
		{"wrong type", `{"name": 1}`, "cannot unmarshal number"},
		{"trailing data", `{} {}`, "unexpected data after the top-level object"},
		{"unknown kind", `{"kind": "server"}`, `kind "server" is not one of`},
		{"unknown tool", `{"tools": {"typos": {}}}`, "tools.typos: unknown tool"},
		{"null tool", `{"tools": {"golangci": null}}`, "tools.golangci: must be an object"},
		{"list tool", `{"tools": {"misspell": []}}`, "tools.misspell: must be an object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, tt.json)
			_, err := Load(dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestLoadUnreadable(t *testing.T) {
	dir := t.TempDir()
	// A directory where the file should be cannot be read as a file.
	if err := os.Mkdir(filepath.Join(dir, FileName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Error("Load of an unreadable config succeeded")
	}
}

func TestKinds(t *testing.T) {
	for _, k := range Kinds {
		if _, err := Parse([]byte(`{"kind":"` + k + `"}`)); err != nil {
			t.Errorf("kind %q rejected: %v", k, err)
		}
	}
}

func TestOverride(t *testing.T) {
	p, err := Parse([]byte(`{"tools": {"misspell": {"ignore": ["x"]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"ignore": []any{"x"}}
	if got := p.Override("misspell"); !reflect.DeepEqual(got, want) {
		t.Errorf("Override(misspell) = %#v, want %#v", got, want)
	}
	if got := p.Override("golangci"); !reflect.DeepEqual(got, map[string]any{}) {
		t.Errorf("Override of a tool with no override = %#v, want an empty object", got)
	}
}

func write(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
