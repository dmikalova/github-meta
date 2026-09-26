// Package ci holds the shared mage targets every Go project runs (ADR 0002,
// 0003). A project imports it from its magefile:
//
//	//go:build mage
//
//	package main
//
//	import (
//		// mage:import ci
//		"github.com/dmikalova/project-standards/ci"
//	)
//
//	func init() {
//		ci.CoverGates = []ci.CoverGate{{Name: "engine", Test: "./internal/engine/", Count: "./internal/engine/"}}
//	}
//
// which gives `mage ci:fix`, `mage ci:check` and one target per check, such as
// `mage ci:lint`. Fix applies every autofix and regenerates the generated
// configs; Check only verifies and never writes a file. The documented local
// validator is `mage ci:fix && mage ci:check`.
//
// Every tool is run with `go run pkg@version`, pinned in tools.go, so none of
// them enter the project's dependency graph.
//
// ci:check includes ci:vuln, so a new vulnerability advisory fails the next
// commit (govulncheck needs network access for its database).
//
// # Environment
//
// CI_COMMIT_RANGE, as "<from>..<to>", makes ci:commits lint exactly that range
// instead of the commits since the merge-base with the default branch. CI sets
// it on a push, from ${{ github.event.before }}..${{ github.sha }}, so commits
// pushed straight to the default branch are linted too. A from of all zeros
// (the first push of a new branch) lints from the merge-base. The clone needs
// full history for either end to resolve.
//
// Exported functions in this package are mage targets, so helpers stay
// unexported. Settings are package variables a project's magefile sets in
// init(); each defaults to none.
package ci
