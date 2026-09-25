package ci

import "context"

// ExtraBuilds are build steps Build runs after `go build ./...`, for targets
// the host build misses, such as a js/wasm client. Each one should build
// without writing into the tree, so Check stays read-only. None by default.
var ExtraBuilds []func(context.Context) error

// CoverGates lists the areas Cover holds at 100% statement coverage. None by
// default, so Cover passes until a project opts packages in.
var CoverGates []CoverGate

// A CoverGate is one area held at 100% coverage. Test names the packages whose
// tests run and Count names the packages whose statements are counted. They
// differ when a package is exercised through another package's tests; both
// take go package patterns such as ./internal/engine/ or ./internal/cards/....
type CoverGate struct {
	// Name labels the gate in Cover's report.
	Name string
	// Test is the package pattern passed to go test.
	Test string
	// Count is the package pattern passed as -coverpkg.
	Count string
}
