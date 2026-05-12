package parser

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	gherkin "github.com/cucumber/gherkin/go/v26"
	messages "github.com/cucumber/messages/go/v21"
)

// Parse reads and parses a single .feature file at path.
func Parse(path string) (*Feature, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open feature file %q: %w", path, err)
	}
	defer f.Close()
	return ParseReader(path, f)
}

// ParseReader parses Gherkin content from r, using uri as the document URI.
func ParseReader(uri string, r io.Reader) (*Feature, error) {
	gen := &messages.Incrementing{}
	doc, err := gherkin.ParseGherkinDocument(r, gen.NewId)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", uri, err)
	}
	doc.Uri = uri

	pickles := gherkin.Pickles(*doc, uri, gen.NewId)
	return buildFeature(doc, pickles), nil
}

// ParseGlob discovers and parses all .feature files matching the glob pattern.
// The pattern is evaluated relative to the process working directory.
// If no files match, a nil error and empty slice are returned.
func ParseGlob(pattern string) ([]*Feature, error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("feature glob %q: %w", pattern, err)
	}
	out := make([]*Feature, 0, len(paths))
	for _, p := range paths {
		feat, err := Parse(p)
		if err != nil {
			return nil, err
		}
		out = append(out, feat)
	}
	return out, nil
}

// buildFeature converts the raw gherkin AST + expanded pickles into our
// internal Feature model. Pickles already handle Scenario Outline expansion.
func buildFeature(doc *messages.GherkinDocument, pickles []*messages.Pickle) *Feature {
	feat := &Feature{URI: doc.Uri}
	if doc.Feature == nil {
		return feat
	}

	mf := doc.Feature
	feat.Name = mf.Name
	feat.Description = strings.TrimSpace(mf.Description)
	feat.Language = mf.Language
	feat.Tags = collectTags(mf.Tags)

	// Background: find the first Background child, if any.
	for _, child := range mf.Children {
		if child.Background != nil {
			feat.Background = buildBackground(child.Background)
			break
		}
	}

	// Build a map from scenario AST id → source line for SourceLine annotation.
	lineMap := buildLineMap(mf.Children)

	// Convert each Pickle into a Scenario. Pickles are fully expanded
	// (Scenario Outline rows have already been substituted).
	feat.Scenarios = make([]*Scenario, 0, len(pickles))
	nameCount := make(map[string]int)
	for _, p := range pickles {
		idx := nameCount[p.Name]
		nameCount[p.Name]++
		sc := &Scenario{
			DeterministicID: deterministicID(doc.Uri, p.Name, idx),
			Name:            p.Name,
			Tags:            collectPickleTags(p.Tags),
			SourceLine:      lineMap[astNodeID(p)],
		}
		sc.Steps = buildSteps(p.Steps)
		feat.Scenarios = append(feat.Scenarios, sc)
	}
	return feat
}

func buildBackground(bg *messages.Background) *Background {
	b := &Background{Name: bg.Name}
	for _, s := range bg.Steps {
		b.Steps = append(b.Steps, &Step{
			Keyword:   s.Keyword,
			Text:      s.Text,
			DocString: convertDocString(s.DocString),
			DataTable: convertDataTable(s.DataTable),
		})
	}
	return b
}

func buildSteps(psteps []*messages.PickleStep) []*Step {
	out := make([]*Step, 0, len(psteps))
	for _, ps := range psteps {
		step := &Step{
			Keyword: pickleStepKeyword(ps.Type),
			Text:    ps.Text,
		}
		if ps.Argument != nil {
			if ps.Argument.DocString != nil {
				step.DocString = &DocString{
					MediaType: ps.Argument.DocString.MediaType,
					Content:   ps.Argument.DocString.Content,
				}
			}
			if ps.Argument.DataTable != nil {
				step.DataTable = convertPickleTable(ps.Argument.DataTable)
			}
		}
		out = append(out, step)
	}
	return out
}

func convertDocString(ds *messages.DocString) *DocString {
	if ds == nil {
		return nil
	}
	return &DocString{MediaType: ds.MediaType, Content: ds.Content}
}

func convertDataTable(dt *messages.DataTable) *DataTable {
	if dt == nil {
		return nil
	}
	return &DataTable{Rows: tableRows(dt.Rows)}
}

func convertPickleTable(pt *messages.PickleTable) *DataTable {
	if pt == nil {
		return nil
	}
	rows := make([][]string, len(pt.Rows))
	for i, row := range pt.Rows {
		cells := make([]string, len(row.Cells))
		for j, cell := range row.Cells {
			cells[j] = cell.Value
		}
		rows[i] = cells
	}
	return &DataTable{Rows: rows}
}

func tableRows(rows []*messages.TableRow) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		cells := make([]string, len(row.Cells))
		for j, cell := range row.Cells {
			cells[j] = cell.Value
		}
		out[i] = cells
	}
	return out
}

func collectTags(tags []*messages.Tag) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, t.Name) // already includes "@"
	}
	return out
}

func collectPickleTags(tags []*messages.PickleTag) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, t.Name)
	}
	return out
}

// buildLineMap returns a map from the first AST node id in a Scenario to its
// source line number, used to annotate Pickles with source location.
func buildLineMap(children []*messages.FeatureChild) map[string]int64 {
	m := make(map[string]int64)
	for _, child := range children {
		if child.Scenario != nil {
			sc := child.Scenario
			if sc.Location != nil {
				m[sc.Id] = sc.Location.Line
			}
		}
	}
	return m
}

// astNodeID returns the first AstNodeId of a Pickle (the scenario node).
func astNodeID(p *messages.Pickle) string {
	if len(p.AstNodeIds) == 0 {
		return ""
	}
	return p.AstNodeIds[0]
}

// deterministicID computes a stable 16-char hex ID for a scenario.
func deterministicID(uri, name string, idx int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", uri, name, idx)))
	return fmt.Sprintf("%x", h[:8])
}

// pickleStepKeyword converts a PickleStepType to a human-readable keyword.
func pickleStepKeyword(t messages.PickleStepType) string {
	switch t {
	case messages.PickleStepType_CONTEXT:
		return "Given "
	case messages.PickleStepType_ACTION:
		return "When "
	case messages.PickleStepType_OUTCOME:
		return "Then "
	default:
		return "* "
	}
}
