package ci

import (
	// The base golangci-lint config runs gocritic's ruleguard check over the
	// generated .ruleguard.go, and ruleguard can only load that file when its
	// dsl package resolves from the project's own module. Importing dsl here
	// puts it in every importing project's module graph, so go mod tidy keeps
	// it. The package is a small DSL with no dependencies and no side effects.
	_ "github.com/quasilyte/go-ruleguard/dsl"
)
