// Command project-standards will run the shared checks in projects that do not
// use Go (ADR 0002). It will be built from the ci package's targets with
// `mage -compile` on each release; until then it only says so.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "project-standards: not built yet; Go projects run `mage ci:check`")
	os.Exit(1)
}
