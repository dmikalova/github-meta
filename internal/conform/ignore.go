package conform

import (
	"bytes"

	"github.com/dmikalova/project-standards/internal/config"
	"github.com/dmikalova/project-standards/internal/generate"
	"github.com/dmikalova/project-standards/templates"
)

// ignoreFiles returns the universal .gitignore and .dockerignore, each with the
// project's additions from mklv.config.json in a marked section at the end.
// They are the same for every project, so a language change never drifts
// them.
func ignoreFiles(p *config.Project) []generate.File {
	var ignore config.Ignore
	if p.Ignore != nil {
		ignore = *p.Ignore
	}
	return []generate.File{
		{Path: gitignorePath, Content: withAdditions(templates.Gitignore, "git", ignore.Git)},
		{
			Path:    dockerignorePath,
			Content: withAdditions(templates.Dockerignore, "docker", ignore.Docker),
		},
	}
}

// withAdditions appends a project's entries under ignore.<key> to a universal
// ignore file. Without any, the file is the universal one alone.
func withAdditions(base []byte, key string, additions []string) []byte {
	if len(additions) == 0 {
		return base
	}
	var buf bytes.Buffer
	buf.Write(base)
	buf.WriteString("\n# This project's additions, from ignore." + key +
		" in mklv.config.json. Edit them there.\n")
	for _, a := range additions {
		buf.WriteString(a + "\n")
	}
	return buf.Bytes()
}
