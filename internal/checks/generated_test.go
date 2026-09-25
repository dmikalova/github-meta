package checks

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// generatedPaths lists the generated configs in the current directory.
func generatedPaths(t *testing.T) []string {
	t.Helper()
	var paths []string
	for _, name := range []string{
		".commitlint.yaml", ".gitleaks.toml", ".golangci.yaml",
		".markdownlint-cli2.yaml", ".ruleguard.go",
	} {
		if _, err := os.Stat(name); err == nil {
			paths = append(paths, name)
		}
	}
	return paths
}

func TestWriteGeneratedAndDrift(t *testing.T) {
	for _, tt := range []struct {
		name     string
		goModule bool
		want     []string
	}{
		{"go module", true, []string{
			".commitlint.yaml", ".gitleaks.toml", ".golangci.yaml",
			".markdownlint-cli2.yaml", ".ruleguard.go",
		}},
		{"no go module", false, []string{
			".commitlint.yaml", ".gitleaks.toml", ".markdownlint-cli2.yaml",
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tt.goModule {
				writeFiles(t, map[string]string{"go.mod": "module example.com/x\n"})
			}
			wantErr(t, testChecks.Drift(), ".commitlint.yaml (missing)")
			var err error
			out := captureStdout(t, func() { err = testChecks.WriteGenerated() })
			if err != nil {
				t.Fatal(err)
			}
			if got := generatedPaths(t); !slices.Equal(got, tt.want) {
				t.Errorf("generated %v, want %v", got, tt.want)
			}
			if n := strings.Count(out, "test fix: writing "); n != len(tt.want) {
				t.Errorf("output = %q, want %d writes", out, len(tt.want))
			}
			if err := testChecks.Drift(); err != nil {
				t.Errorf("Drift after WriteGenerated = %v", err)
			}
			// Up-to-date files are left alone.
			out = captureStdout(t, func() { err = testChecks.WriteGenerated() })
			if err != nil || out != "" {
				t.Errorf("second WriteGenerated = %v, printing %q; want nothing written", err, out)
			}
			// A hand edit is drift, and the fix command is named.
			writeFiles(t, map[string]string{".gitleaks.toml": "# edited\n"})
			err = testChecks.Drift()
			wantErr(t, err, ".gitleaks.toml (differs)")
			wantErr(t, err, "run test fix")
		})
	}
}

func TestDriftIgnoresGoConfigsWithoutGoModule(t *testing.T) {
	t.Chdir(t.TempDir())
	captureStdout(t, func() {
		if err := testChecks.WriteGenerated(); err != nil {
			t.Fatal(err)
		}
	})
	writeFiles(t, map[string]string{".golangci.yaml": "stale\n", ".ruleguard.go": "stale\n"})
	if err := testChecks.Drift(); err != nil {
		t.Errorf("Drift = %v, want the Go-only configs ignored without a go.mod", err)
	}
}

func TestGeneratedErrors(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{"mklv.config.json": `{"kind":"nope"}`})
	wantErr(t, testChecks.Drift(), `kind "nope"`)
	wantErr(t, testChecks.WriteGenerated(), `kind "nope"`)

	writeFiles(t, map[string]string{
		"mklv.config.json": `{"tools":{"gitleaks":{"x":[null]}}}`,
	})
	wantErr(t, testChecks.Drift(), "encoding .gitleaks.toml")

	// A directory where a generated config belongs can be neither read nor
	// written.
	writeFiles(t, map[string]string{"mklv.config.json": `{}`, ".commitlint.yaml/x": ""})
	wantErr(t, testChecks.Drift(), "is a directory")
	captureStdout(t, func() {
		wantErr(t, testChecks.WriteGenerated(), "is a directory")
	})
}
