// Package coverage implements surface-area scanning and coverage reporting for
// the lobster coverage command. It parses proto files, an OpenAPI spec, and a
// hardcoded CLI command list to build the known API/CLI surface, then compares
// that surface against feature file @covers:* tags and inferred URL/command
// patterns to produce a coverage report.
package coverage

// SurfaceKind categorises what kind of thing a CoverageItem represents.
type SurfaceKind string

const (
	KindRPC  SurfaceKind = "rpc"
	KindHTTP SurfaceKind = "http"
	KindCLI  SurfaceKind = "cli"
)

// CoverageItem represents one unit of the surface area to be covered.
type CoverageItem struct {
	// ID is the canonical identifier used in @covers: tags.
	// Examples: "RunService.RunAsync", "GET:/api/v1/runs", "cli:run"
	ID string

	// Kind is the category of this item.
	Kind SurfaceKind

	// Label is a human-readable description.
	Label string

	// Service is the gRPC service name (KindRPC only).
	Service string

	// Method is the gRPC method name (KindRPC only).
	Method string

	// HTTPMethod is the HTTP verb (KindHTTP only).
	HTTPMethod string

	// HTTPPath is the HTTP path (KindHTTP only).
	HTTPPath string

	// CLICommand is the full command path (KindCLI only), e.g. "run:watch".
	CLICommand string
}

// CoverageResult pairs a CoverageItem with its observed coverage data.
type CoverageResult struct {
	Item CoverageItem

	// ExplicitFeatures lists feature file URIs that carry an @covers:<ID> tag.
	ExplicitFeatures []string

	// InferredFeatures lists feature file URIs where URL/command patterns
	// matching this item were detected in step text (lower-confidence signal).
	InferredFeatures []string

	// ScenarioCount is the total number of scenarios across all explicit feature
	// files that explicitly cover this item.
	ScenarioCount int

	// Covered is true when at least one explicit feature covers this item.
	Covered bool

	// BelowWarnThreshold is true when ScenarioCount < warnMinScenarios.
	BelowWarnThreshold bool

	// BelowMinThreshold is true when ScenarioCount < minScenarios.
	BelowMinThreshold bool
}

// Report is the complete output of a coverage analysis.
type Report struct {
	Items []CoverageResult

	TotalSurface   int
	CoveredCount   int
	UncoveredCount int

	// ThresholdViolations counts items that violate --min-scenarios.
	ThresholdViolations int

	// WarnViolations counts items below --warn-min-scenarios (but above minScenarios).
	WarnViolations int

	MinScenarios     int
	WarnMinScenarios int
}

// BuildReport combines surface items with coverage data and thresholds.
func BuildReport(
	surface []CoverageItem,
	explicit map[string][]string, // coverID → []featureURI
	scenarioCounts map[string]int, // coverID → scenario count
	inferred map[string][]string, // coverID → []featureURI
	minScenarios int,
	warnMinScenarios int,
) Report {
	r := Report{
		TotalSurface:     len(surface),
		MinScenarios:     minScenarios,
		WarnMinScenarios: warnMinScenarios,
	}

	for _, item := range surface {
		explicitFeats := explicit[item.ID]
		inferredFeats := inferred[item.ID]
		count := scenarioCounts[item.ID]

		covered := len(explicitFeats) > 0

		belowMin := covered && count < minScenarios
		belowWarn := covered && count < warnMinScenarios && !belowMin

		res := CoverageResult{
			Item:               item,
			ExplicitFeatures:   explicitFeats,
			InferredFeatures:   inferredFeats,
			ScenarioCount:      count,
			Covered:            covered,
			BelowWarnThreshold: belowWarn,
			BelowMinThreshold:  belowMin,
		}
		r.Items = append(r.Items, res)

		if covered {
			r.CoveredCount++
		} else {
			r.UncoveredCount++
		}
		if belowMin {
			r.ThresholdViolations++
		}
		if belowWarn {
			r.WarnViolations++
		}
	}

	return r
}
