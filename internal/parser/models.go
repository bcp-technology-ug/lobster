// Package parser provides Gherkin feature-file parsing and linting for Lobster.
// It converts raw .feature files into the Feature/Scenario/Step internal model
// used by the runner, step registry, and planner.
package parser

// Feature is a parsed Gherkin feature file.
type Feature struct {
	// URI is the file path or URI that was parsed.
	URI         string
	Name        string
	Description string
	Tags        []string
	Language    string
	Background  *Background
	Scenarios   []*Scenario
}

// Background holds steps that run before each scenario in the feature.
type Background struct {
	Name  string
	Steps []*Step
}

// Scenario is one executable scenario. Scenario Outlines are expanded into
// individual Scenario values (one per Examples row) by the parser.
type Scenario struct {
	// DeterministicID is a stable hex string identifying this scenario. It is
	// computed as the first 16 hex chars of SHA-256(URI:Name:RowIndex).
	DeterministicID string
	Name            string
	Description     string
	Tags            []string
	Steps           []*Step
	// SourceLine is the 1-based line number of the Scenario keyword.
	SourceLine int64
}

// Step is a single Given / When / Then / And / But line.
type Step struct {
	// Keyword is the step keyword including trailing space, e.g. "Given ".
	Keyword   string
	Text      string
	DocString *DocString
	DataTable *DataTable
}

// DocString is an inline triple-quoted text block attached to a step.
type DocString struct {
	MediaType string
	Content   string
}

// DataTable is a pipe-delimited table argument attached to a step.
// Rows[0] is the header row when the step semantics treat it as such.
type DataTable struct {
	Rows [][]string // [row][col]
}
