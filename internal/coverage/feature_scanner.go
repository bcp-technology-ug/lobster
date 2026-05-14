package coverage

import (
	"regexp"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/parser"
)

// coversTagPrefix is the tag prefix used to declare explicit coverage.
const coversTagPrefix = "@covers:"

// urlInferPattern matches HTTP paths like /api/v1/... in step text.
var urlInferPattern = regexp.MustCompile(`/api/v1/[^\s"']+`)

// cliInferPattern matches lobster subcommand invocations in step text.
var cliInferPattern = regexp.MustCompile(`(?i)\blobster\s+(run(?:\s+(?:watch|status|cancel))?|validate|lint|plan|init|config|runs|plans|stack|integrations|admin|doctor|tui|coverage)\b`)

// FeatureCoverageData holds the results of scanning feature files.
type FeatureCoverageData struct {
	// Explicit maps coverageItemID → []featureURI (from @covers: tags)
	Explicit map[string][]string

	// ScenarioCounts maps coverageItemID → total scenario count across all
	// feature files that explicitly cover it.
	ScenarioCounts map[string]int

	// Inferred maps coverageItemID → []featureURI (from URL/command scan)
	Inferred map[string][]string
}

// ScanFeatureGlob scans all .feature files matching glob for coverage signals.
func ScanFeatureGlob(glob string, surface []CoverageItem) (*FeatureCoverageData, error) {
	paths, err := expandGlob(glob)
	if err != nil {
		return nil, err
	}

	data := &FeatureCoverageData{
		Explicit:       make(map[string][]string),
		ScenarioCounts: make(map[string]int),
		Inferred:       make(map[string][]string),
	}

	// Build a lookup set of known IDs for inference matching.
	knownHTTPPaths := make(map[string]string) // normalised path → item ID
	knownCLICmds := make(map[string]string)   // lower cmd name → item ID
	for _, item := range surface {
		switch item.Kind {
		case KindHTTP:
			knownHTTPPaths[item.HTTPPath] = item.ID
		case KindCLI:
			// Extract the base subcommand, e.g. "run:watch" → "run watch"
			cmd := strings.ReplaceAll(strings.TrimPrefix(item.CLICommand, "cli:"), ":", " ")
			knownCLICmds[strings.ToLower(cmd)] = item.ID
		}
	}

	for _, p := range paths {
		feat, err := parser.Parse(p)
		if err != nil {
			// Skip unparseable files — the user will learn from validate/lint
			continue
		}

		uri := p

		// --- Explicit coverage from @covers: tags ---
		for _, tag := range feat.Tags {
			if strings.HasPrefix(tag, coversTagPrefix) {
				id := strings.TrimPrefix(tag, coversTagPrefix)
				data.Explicit[id] = appendUnique(data.Explicit[id], uri)
				data.ScenarioCounts[id] += len(feat.Scenarios)
			}
		}

		// --- Inferred coverage from step text ---
		inferredIDs := make(map[string]bool)
		for _, sc := range feat.Scenarios {
			for _, step := range sc.Steps {
				// URL inference
				for _, match := range urlInferPattern.FindAllString(step.Text, -1) {
					// Try each HTTP item to see if the path is a prefix/match
					for path, id := range knownHTTPPaths {
						if strings.HasPrefix(match, path) || match == path {
							inferredIDs[id] = true
						}
					}
				}
				// CLI inference
				cliMatches := cliInferPattern.FindAllStringSubmatch(step.Text, -1)
				for _, m := range cliMatches {
					if len(m) > 1 {
						cmd := strings.ToLower(strings.Join(strings.Fields(m[1]), " "))
						// Try longest match first by checking sub-commands
						for knownCmd, id := range knownCLICmds {
							if strings.HasPrefix(cmd, knownCmd) || knownCmd == cmd {
								inferredIDs[id] = true
							}
						}
					}
				}
			}
		}
		for id := range inferredIDs {
			// Only add as inferred if not already explicitly covered by this file.
			alreadyExplicit := false
			for _, eu := range data.Explicit[id] {
				if eu == uri {
					alreadyExplicit = true
					break
				}
			}
			if !alreadyExplicit {
				data.Inferred[id] = appendUnique(data.Inferred[id], uri)
			}
		}
	}

	return data, nil
}

func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}
