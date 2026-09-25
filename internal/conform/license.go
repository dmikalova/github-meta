package conform

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/dmikalova/project-standards/internal/generate"
	"github.com/dmikalova/project-standards/templates"
)

var licenseTemplate = template.Must(template.New(licensePath).Parse(templates.License))

// license renders LICENSE with the years the project has commits in.
func (c *conformer) license() (generate.File, error) {
	shallow, err := c.run(c.dir, "git", "rev-parse", "--is-shallow-repository")
	if err != nil {
		return generate.File{}, err
	}
	if string(bytes.TrimSpace(shallow)) == "true" {
		return generate.File{}, errors.New("the checkout is shallow, and LICENSE's years " +
			"come from the full commit history; clone with full history (fetch-depth: 0)")
	}
	out, err := c.run(c.dir, "git", "log", "--format=%ad", "--date=format:%Y")
	if err != nil {
		return generate.File{}, err
	}
	years, err := parseYears(string(out))
	if err != nil {
		return generate.File{}, err
	}
	return generate.File{Path: licensePath, Content: renderLicense(yearRanges(years), Owner)}, nil
}

// parseYears reads git log's one year per line.
func parseYears(out string) ([]int, error) {
	var years []int
	for line := range strings.FieldsSeq(out) {
		y, err := strconv.Atoi(line)
		if err != nil {
			return nil, fmt.Errorf("git log: %q is not a year", line)
		}
		years = append(years, y)
	}
	if len(years) == 0 {
		return nil, errors.New("git log: the project has no commits")
	}
	return years, nil
}

// yearRanges lists years in order, collapsing consecutive ones into a range:
// 2021, 2023-2026.
func yearRanges(years []int) string {
	years = slices.Clone(years)
	slices.Sort(years)
	years = slices.Compact(years)
	var parts []string
	for i := 0; i < len(years); {
		j := i
		for j+1 < len(years) && years[j+1] == years[j]+1 {
			j++
		}
		part := strconv.Itoa(years[i])
		if j > i {
			part += "-" + strconv.Itoa(years[j])
		}
		parts = append(parts, part)
		i = j + 1
	}
	return strings.Join(parts, ", ")
}

// renderLicense fills the appendix copyright line of the Apache-2.0 text.
func renderLicense(years, owner string) []byte {
	var buf bytes.Buffer
	// Executing a parsed template with two string fields into a buffer cannot
	// fail.
	_ = licenseTemplate.Execute(&buf, struct{ Years, Owner string }{years, owner})
	return buf.Bytes()
}
