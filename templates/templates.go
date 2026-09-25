// Package templates holds the base configs and standard files that
// project-standards writes into projects, embedded so the ci package and the
// binary carry them without reading this repository from disk.
package templates

import "embed"

// Base holds one base config per tool under base/, in the tool's own format.
// A project's overrides from mklv.config.json are merged onto these (ADR 0005).
//
//go:embed base
var Base embed.FS

// Ruleguard is the ruleguard ruleset gocritic loads through the base
// golangci-lint config. It is written into each project verbatim.
//
//go:embed ruleguard/rules.go
var Ruleguard []byte
