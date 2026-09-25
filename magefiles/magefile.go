//go:build mage

// project-standards dogfoods its own shared targets. Run `mage -l` to list them,
// and `mage ci:fix && mage ci:check` before calling work done.
package main

import (
	// mage:import ci
	"github.com/dmikalova/project-standards/ci"
)

func init() {
	ci.CoverGates = []ci.CoverGate{
		{Name: "changelog", Test: "./internal/changelog/", Count: "./internal/changelog/"},
		{Name: "checks", Test: "./internal/checks/", Count: "./internal/checks/"},
		{Name: "conform", Test: "./internal/conform/", Count: "./internal/conform/"},
		{Name: "config", Test: "./internal/config/", Count: "./internal/config/"},
		{Name: "generate", Test: "./internal/generate/", Count: "./internal/generate/"},
	}
}
