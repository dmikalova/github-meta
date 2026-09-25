package conform

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/dmikalova/project-standards/internal/config"
	"github.com/dmikalova/project-standards/schema"
)

// config loads mklv.config.json. An invalid file is reported, and it returns
// nil: the generated configs and ignore files depend on it, and writing them
// from a config it cannot trust could drop the project's overrides.
func (c *conformer) config() (*config.Project, error) {
	data, ok, err := c.read(config.FileName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &config.Project{}, nil
	}
	problems, err := validate(configSchema, data)
	if err != nil {
		return nil, fmt.Errorf("compiling the mklv.config.json schema: %w", err)
	}
	var p *config.Project
	if len(problems) == 0 {
		// The schema and config.Parse should agree; if they do not, the file
		// is reported the same way.
		if p, err = config.Parse(data); err != nil {
			problems = []string{err.Error()}
		}
	}
	if len(problems) > 0 {
		c.find(ruleConfig, config.FileName, "invalid against schema/mklv.config.schema.json: "+
			strings.Join(problems, "; ")+
			"; the generated configs and ignore files were left as they are until it is fixed")
		return nil, nil
	}
	return p, nil
}

// configSchema is the schema mklv.config.json is validated against. Tests
// replace it to cover a schema that does not compile.
var configSchema = schema.MklvConfig

// schemaURL names the schema within the compiler. Nothing is fetched.
const schemaURL = "mklv.config.schema.json"

// validate checks a mklv.config.json against the JSON Schema in schemaJSON and
// returns each violation as "<location>: <problem>", sorted. It fails only
// when the schema itself does not compile.
func validate(schemaJSON, data []byte) ([]string, error) {
	compiler := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err == nil {
		err = compiler.AddResource(schemaURL, doc)
	}
	if err != nil {
		return nil, err
	}
	sch, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, err
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return []string{"not valid JSON: " + err.Error()}, nil
	}
	var verr *jsonschema.ValidationError
	if !errors.As(sch.Validate(instance), &verr) {
		return nil, nil
	}
	var problems []string
	collectProblems(verr, &problems)
	slices.Sort(problems)
	return slices.Compact(problems), nil
}

// printer renders schema violations in English.
var printer = message.NewPrinter(language.English)

// pointerEscaper escapes a JSON Pointer token.
var pointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// collectProblems appends each leaf violation under e, the ones that say what
// is actually wrong, as "<JSON Pointer>: <problem>".
func collectProblems(e *jsonschema.ValidationError, problems *[]string) {
	if len(e.Causes) == 0 {
		var loc strings.Builder
		for _, token := range e.InstanceLocation {
			loc.WriteString("/" + pointerEscaper.Replace(token))
		}
		if loc.Len() == 0 {
			loc.WriteString("/")
		}
		*problems = append(*problems, loc.String()+": "+e.ErrorKind.LocalizedString(printer))
	}
	for _, cause := range e.Causes {
		collectProblems(cause, problems)
	}
}
