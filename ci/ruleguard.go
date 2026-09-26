package ci

import (
	// The base golangci-lint config runs gocritic's ruleguard check over the
	// generated .ruleguard.go, and ruleguard can only load that file when every
	// package it imports resolves from the project's own module. The file has
	// the ignore build tag, so go mod tidy never sees its imports. Importing
	// them here puts them in every importing project's module graph, so tidy
	// keeps them.
	//
	// dsl is the ruleguard DSL. rules and uber-rules are the rule bundles the
	// ruleset imports with dsl.ImportRules; their packages hold only rule
	// functions over dsl and have no side effects. Their go.mod files require
	// go-ruleguard and golang.org/x/tools, which therefore join each project's
	// module graph, though no package from either is compiled.
	_ "github.com/quasilyte/go-ruleguard/dsl"
	_ "github.com/quasilyte/go-ruleguard/rules"
	_ "github.com/quasilyte/uber-rules"
)
