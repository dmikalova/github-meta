package ci

// Secrets scans history, staged and unstaged changes for secrets.
// It runs gitleaks with the generated .gitleaks.toml, redacting what it finds.
// Covering all three means a secret is caught before it is committed (the
// pre-commit hook runs ci:check) and cannot hide in an older commit. Every
// scan runs even after one finds a leak, so one run reports them all.
func Secrets() error {
	return shared.Secrets()
}
