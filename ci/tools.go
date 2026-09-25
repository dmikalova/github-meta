package ci

// Every tool the targets run, pinned as a `go run` package@version. These are
// the only version pins for the shared checks (ADR 0004): bumping one here
// bumps every project on its next project-standards update.
const (
	// mage is the task runner itself. Projects run it with `go run`, and ci's own
	// import of the mage library is held to the same version (see
	// TestMageVersionMatchesGoMod).
	mage = "github.com/magefile/mage@v1.17.2"

	// golangciLint is the Go linter.
	golangciLint = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0"

	// commitlint is the Go port of commitlint, which checks commit messages.
	commitlint = "github.com/conventionalcommit/commitlint@v0.12.0"

	// misspell checks spelling against a list of known misspellings.
	misspell = "github.com/golangci/misspell/cmd/misspell@v0.8.0"

	// goldmarkLint is a Go port of markdownlint.
	goldmarkLint = "github.com/mrueg/goldmark-lint/cmd/goldmark-lint@v0.5.3"

	// gitleaks finds secrets. The module declares the zricethezav path, so the
	// github.com/gitleaks path fails with go run.
	gitleaks = "github.com/zricethezav/gitleaks/v8@v8.30.1"

	// golines applies gofmt's formatting and shortens long lines.
	golines = "github.com/golangci/golines@v0.15.0"

	// goimports repairs imports after golangci-lint --fix: ruleguard's
	// suggested rewrites can introduce a package (bytes, io) without importing
	// it, which would leave ci:fix with a tree that does not build.
	goimports = "golang.org/x/tools/cmd/goimports@v0.50.0"

	// gci groups imports into standard, third-party and local-module sections,
	// which gofmt and golines do not do.
	gci = "github.com/daixiang0/gci@v0.14.0"
)
