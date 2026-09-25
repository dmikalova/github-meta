// Package schema embeds the JSON Schema of mklv.config.json, so the binary can
// validate a project's config without reading this repository from disk.
package schema

import _ "embed"

// MklvConfig is schema/mklv.config.schema.json, the schema editors validate
// mklv.config.json against.
//
//go:embed mklv.config.schema.json
var MklvConfig []byte
