package ci

// Drift fails if a generated config differs from what ci:fix writes.
// Generated configs are the base configs merged with mklv.config.json; a hand
// edit to one is drift, and the fix is to move the change into
// mklv.config.json.
func Drift() error {
	return shared.Drift()
}
