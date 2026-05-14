package coverage

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/ui"
)

// RenderText writes a human-readable coverage report to w.
func RenderText(w io.Writer, r Report) {
	// Group results by kind.
	byKind := map[SurfaceKind][]CoverageResult{}
	for _, res := range r.Items {
		byKind[res.Item.Kind] = append(byKind[res.Item.Kind], res)
	}

	kindOrder := []SurfaceKind{KindRPC, KindHTTP, KindCLI}
	kindLabels := map[SurfaceKind]string{
		KindRPC:  "gRPC / RPC Methods",
		KindHTTP: "HTTP Endpoints",
		KindCLI:  "CLI Commands",
	}

	for _, kind := range kindOrder {
		items := byKind[kind]
		if len(items) == 0 {
			continue
		}
		// Sort by ID for stable output.
		sort.Slice(items, func(i, j int) bool {
			return items[i].Item.ID < items[j].Item.ID
		})

		_, _ = fmt.Fprintln(w, ui.StyleSubheading.Render(kindLabels[kind]))
		for _, res := range items {
			var icon, status string
			switch {
			case !res.Covered:
				icon = ui.StyleError.Render(ui.IconCross)
				status = ui.StyleError.Render("uncovered")
			case res.BelowMinThreshold:
				icon = ui.StyleError.Render(ui.IconWarning)
				status = ui.StyleError.Render(fmt.Sprintf("%d scenario(s) — below min", res.ScenarioCount))
			case res.BelowWarnThreshold:
				icon = ui.StyleWarning.Render(ui.IconWarning)
				status = ui.StyleWarning.Render(fmt.Sprintf("%d scenario(s) — below warn threshold", res.ScenarioCount))
			default:
				icon = ui.StyleSuccess.Render(ui.IconCheck)
				status = ui.StyleSuccess.Render(fmt.Sprintf("%d scenario(s)", res.ScenarioCount))
			}

			_, _ = fmt.Fprintf(w, "  %s  %-60s %s\n",
				icon,
				ui.StyleCode.Render(res.Item.ID),
				status,
			)

			// Show inferred-only note.
			if !res.Covered && len(res.InferredFeatures) > 0 {
				_, _ = fmt.Fprintf(w, "       %s  inferred from: %s\n",
					ui.StyleMuted.Render(ui.IconInfo),
					ui.StyleMuted.Render(strings.Join(shortPaths(res.InferredFeatures), ", ")),
				)
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	// Summary line.
	covPct := 0
	if r.TotalSurface > 0 {
		covPct = (r.CoveredCount * 100) / r.TotalSurface
	}

	summaryStyle := ui.StyleSuccess
	if r.UncoveredCount > 0 || r.ThresholdViolations > 0 {
		summaryStyle = ui.StyleError
	} else if r.WarnViolations > 0 {
		summaryStyle = ui.StyleWarning
	}

	_, _ = fmt.Fprintln(w, summaryStyle.Render(fmt.Sprintf(
		"Coverage: %d/%d (%d%%) — %d uncovered, %d below min, %d below warn",
		r.CoveredCount, r.TotalSurface, covPct,
		r.UncoveredCount, r.ThresholdViolations, r.WarnViolations,
	)))
}

// jsonCoverageResult is the JSON wire format for a single coverage result.
type jsonCoverageResult struct {
	ID               string   `json:"id"`
	Kind             string   `json:"kind"`
	Label            string   `json:"label"`
	Covered          bool     `json:"covered"`
	ScenarioCount    int      `json:"scenario_count"`
	ExplicitFeatures []string `json:"explicit_features,omitempty"`
	InferredFeatures []string `json:"inferred_features,omitempty"`
	BelowMin         bool     `json:"below_min,omitempty"`
	BelowWarn        bool     `json:"below_warn,omitempty"`
}

// jsonReport is the top-level JSON output structure.
type jsonReport struct {
	TotalSurface        int                  `json:"total_surface"`
	CoveredCount        int                  `json:"covered_count"`
	UncoveredCount      int                  `json:"uncovered_count"`
	ThresholdViolations int                  `json:"threshold_violations"`
	WarnViolations      int                  `json:"warn_violations"`
	MinScenarios        int                  `json:"min_scenarios"`
	WarnMinScenarios    int                  `json:"warn_min_scenarios"`
	Items               []jsonCoverageResult `json:"items"`
}

// RenderJSON writes a machine-readable JSON coverage report to w.
func RenderJSON(w io.Writer, r Report) error {
	items := make([]jsonCoverageResult, 0, len(r.Items))
	for _, res := range r.Items {
		items = append(items, jsonCoverageResult{
			ID:               res.Item.ID,
			Kind:             string(res.Item.Kind),
			Label:            res.Item.Label,
			Covered:          res.Covered,
			ScenarioCount:    res.ScenarioCount,
			ExplicitFeatures: res.ExplicitFeatures,
			InferredFeatures: res.InferredFeatures,
			BelowMin:         res.BelowMinThreshold,
			BelowWarn:        res.BelowWarnThreshold,
		})
	}

	out := jsonReport{
		TotalSurface:        r.TotalSurface,
		CoveredCount:        r.CoveredCount,
		UncoveredCount:      r.UncoveredCount,
		ThresholdViolations: r.ThresholdViolations,
		WarnViolations:      r.WarnViolations,
		MinScenarios:        r.MinScenarios,
		WarnMinScenarios:    r.WarnMinScenarios,
		Items:               items,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// shortPaths trims paths to their base names for compact display.
func shortPaths(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.Base(p)
	}
	return out
}
