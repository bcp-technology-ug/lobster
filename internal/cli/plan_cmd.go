package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	planv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology-ug/lobster/internal/runner"
	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newPlanCommand creates the `lobster plan` command wired to a real Planner.
func newPlanCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate a deterministic execution plan without running steps",
		Long:  "Parse feature files and build an execution plan showing which scenarios will run and in what order.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return planCommand(cmd, v)
		},
	}

	cmd.Flags().String("features", "", "feature file glob pattern (e.g. features/**/*.feature)")
	cmd.Flags().StringSlice("tags", nil, "filter scenarios by tag expression (e.g. @smoke)")
	cmd.Flags().String("scenario-regex", "", "filter scenarios by name regex")
	cmd.Flags().StringSlice("compose-profile", nil, "compose profile(s) to activate")
	cmd.Flags().Bool("persist", false, "persist the plan as an artifact in the store")
	cmd.Flags().String("format", "text", "output format: text|json")
	cmd.Flags().String("out", "", "write plan JSON to file (implies --format json)")
	addPersistenceFlags(cmd.Flags())
	return cmd
}

func planCommand(cmd *cobra.Command, v *viper.Viper) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// --- feature paths ---
	featureFlag, _ := cmd.Flags().GetString("features")
	featuresCfg := v.GetStringSlice("features.paths")
	if featureFlag != "" {
		featuresCfg = []string{featureFlag}
	}
	if len(featuresCfg) == 0 {
		featuresCfg = []string{"features/**/*.feature"}
	}

	// --- tags ---
	tags, _ := cmd.Flags().GetStringSlice("tags")
	tagExpr := strings.Join(tags, ",")
	if tagExpr == "" {
		tagExpr = v.GetString("runner.tags")
	}

	// --- other flags ---
	persist, _ := cmd.Flags().GetBool("persist")
	format, _ := cmd.Flags().GetString("format")
	outFile, _ := cmd.Flags().GetString("out")
	scenarioRegex, _ := cmd.Flags().GetString("scenario-regex")
	if outFile != "" {
		format = "json"
	}

	// --- config provider ---
	baseURL := v.GetString("http.base_url")
	configFn := func(_ context.Context, _, _ string) (*runner.RunConfig, error) {
		return &runner.RunConfig{
			BaseURL:       baseURL,
			FeaturePaths:  featuresCfg,
			Variables:     v.GetStringMapString("variables"),
			ScenarioRegex: scenarioRegex,
		}, nil
	}

	// --- store ---
	storeConfig, err := buildStoreConfigFromInputs(cmd, v)
	if err != nil {
		return fmt.Errorf("store config: %w", err)
	}
	var st *store.Store
	if persist && storeConfig.SQLitePath != "" {
		st, err = store.Open(ctx, storeConfig)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer st.Close()
	}

	// --- planner ---
	planner := runner.NewPlanner(configFn, st)

	featurePath := ""
	if len(featuresCfg) > 0 {
		featurePath = featuresCfg[0]
	}

	req := &planv1.PlanRequest{
		Selector: &runv1.RunSelector{
			FeaturePath:   featurePath,
			TagExpression: tagExpr,
		},
		PersistArtifact: persist,
	}

	resp, err := planner.Plan(ctx, req)
	if err != nil {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Plan failed", err.Error(), "", ""))
		return &ExitError{Code: 1}
	}

	plan := resp.GetPlan()
	if plan == nil || len(plan.GetScenarios()) == 0 {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderInfo("Plan", "No scenarios matched the selector."))
		return nil
	}

	switch format {
	case "json":
		return renderPlanJSON(cmd, plan, featuresCfg, outFile)
	default:
		renderPlanText(cmd, plan, persist)
	}
	return nil
}

func renderPlanText(cmd *cobra.Command, plan *planv1.ExecutionPlan, persisted bool) {
	scenarios := plan.GetScenarios()

	rows := make([][2]string, 0, len(scenarios))
	for i, sc := range scenarios {
		tags := ""
		if len(sc.GetTags()) > 0 {
			tags = " " + strings.Join(sc.GetTags(), " ")
		}
		rows = append(rows, [2]string{
			fmt.Sprintf("%d. %s", i+1, sc.GetFeatureName()),
			sc.GetScenarioName() + tags,
		})
	}

	title := fmt.Sprintf("Execution Plan · %d scenario(s)", len(scenarios))
	if plan.GetPlanId() != "" {
		title += " · plan/" + plan.GetPlanId()[:8]
	}

	_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable(title, rows))

	if persisted && plan.GetPlanId() != "" {
		msg := fmt.Sprintf("Plan persisted with ID %s", plan.GetPlanId())
		_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderSuccess("Plan persisted", msg))
	}
}

func renderPlanJSON(cmd *cobra.Command, plan *planv1.ExecutionPlan, featuresCfg []string, outFile string) error {
	type jsonScenario struct {
		Feature    string   `json:"feature"`
		Scenario   string   `json:"scenario"`
		ScenarioID string   `json:"scenario_id,omitempty"`
		Tags       []string `json:"tags"`
	}
	type jsonPlan struct {
		PlanID      string         `json:"plan_id,omitempty"`
		FeaturePath string         `json:"feature_path,omitempty"`
		ScenarioIDs []string       `json:"scenario_ids,omitempty"`
		Scenarios   []jsonScenario `json:"scenarios"`
		Total       int            `json:"total"`
	}

	scenarios := plan.GetScenarios()
	jScenarios := make([]jsonScenario, 0, len(scenarios))
	scenarioIDs := make([]string, 0, len(scenarios))
	for _, sc := range scenarios {
		jScenarios = append(jScenarios, jsonScenario{
			Feature:    sc.GetFeatureName(),
			Scenario:   sc.GetScenarioName(),
			ScenarioID: sc.GetScenarioId(),
			Tags:       sc.GetTags(),
		})
		if sc.GetScenarioId() != "" {
			scenarioIDs = append(scenarioIDs, sc.GetScenarioId())
		}
	}

	featurePath := ""
	if len(featuresCfg) == 1 {
		featurePath = featuresCfg[0]
	} else if plan.GetSelector() != nil {
		featurePath = plan.GetSelector().GetFeaturePath()
	}

	out := jsonPlan{
		PlanID:      plan.GetPlanId(),
		FeaturePath: featurePath,
		ScenarioIDs: scenarioIDs,
		Scenarios:   jScenarios,
		Total:       len(jScenarios),
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")

	// If --out is specified, write to file instead of (or in addition to) stdout.
	if outFile != "" {
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(outFile, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("write plan file %q: %w", outFile, err)
		}
		_, _ = fmt.Fprint(cmd.OutOrStdout(),
			ui.RenderSuccess("Plan saved", fmt.Sprintf("Written to %s (%d scenario(s))", outFile, len(jScenarios))))
		return nil
	}

	return enc.Encode(out)
}
