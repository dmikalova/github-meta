package ci

// Markdown lints Markdown with goldmark-lint, fixing nothing.
func Markdown() error {
	return shared.Markdown()
}

// Spell checks spelling in Go comments and every other text file.
// It uses misspell with the tools.misspell settings from mklv.config.json and
// fixes nothing.
func Spell() error {
	return shared.Spell(false)
}
