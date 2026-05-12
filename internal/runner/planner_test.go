package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	planv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPlan_nilSelector_returnsInvalidArgument(t *testing.T) {
	t.Parallel()
	p := NewPlanner(emptyCfgFn, nil)
	_, err := p.Plan(context.Background(), &planv1.PlanRequest{})
	if err == nil {
		t.Fatal("expected error for nil selector")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("code: got %v want InvalidArgument", st.Code())
	}
}

func TestPlan_emptyFeaturePaths_returnsEmptyPlan(t *testing.T) {
	t.Parallel()
	p := NewPlanner(emptyCfgFn, nil)
	req := &planv1.PlanRequest{
		Selector: &runv1.RunSelector{
			WorkspaceId: "ws-test",
			ProfileName: "default",
		},
	}
	resp, err := p.Plan(context.Background(), req)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if resp.Plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if resp.Plan.PlanId == "" {
		t.Error("expected non-empty PlanId")
	}
	if resp.Plan.WorkspaceId != "ws-test" {
		t.Errorf("WorkspaceId: got %q want %q", resp.Plan.WorkspaceId, "ws-test")
	}
	if len(resp.Plan.Scenarios) != 0 {
		t.Errorf("expected 0 scenarios, got %d", len(resp.Plan.Scenarios))
	}
}

func TestPlan_withFeatureFile(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Auth
  @smoke
  Scenario: Login succeeds
    Given I have valid credentials

  @wip
  Scenario: Login fails
    Given I have invalid credentials
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "auth.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}}, nil
	}
	p := NewPlanner(cfgFn, nil)
	req := &planv1.PlanRequest{
		Selector: &runv1.RunSelector{
			WorkspaceId: "ws-test",
		},
	}
	resp, err := p.Plan(context.Background(), req)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(resp.Plan.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(resp.Plan.Scenarios))
	}
}

func TestPlan_tagFilter(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Auth
  @smoke
  Scenario: Login succeeds
    Given I have valid credentials

  @wip
  Scenario: Login fails
    Given I have invalid credentials
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "auth.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}}, nil
	}
	p := NewPlanner(cfgFn, nil)
	req := &planv1.PlanRequest{
		Selector: &runv1.RunSelector{
			WorkspaceId:   "ws-test",
			TagExpression: "@smoke",
		},
	}
	resp, err := p.Plan(context.Background(), req)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(resp.Plan.Scenarios) != 1 {
		t.Errorf("expected 1 scenario with @smoke, got %d", len(resp.Plan.Scenarios))
	}
}

func TestPlan_selectorFeaturePathOverride(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Override
  Scenario: Only this one
    Given a step
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "override.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	// Config returns a different path that doesn't exist, but selector.FeaturePath overrides it.
	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{"/nonexistent/path/*.feature"}}, nil
	}
	p := NewPlanner(cfgFn, nil)
	req := &planv1.PlanRequest{
		Selector: &runv1.RunSelector{
			WorkspaceId: "ws-test",
			FeaturePath: fp,
		},
	}
	resp, err := p.Plan(context.Background(), req)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(resp.Plan.Scenarios) != 1 {
		t.Errorf("expected 1 scenario from overridden path, got %d", len(resp.Plan.Scenarios))
	}
}

func TestPlan_scenarioRegexFilter_matchingScenarios(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Filtering
  Scenario: Alpha check
    Given a step

  Scenario: Beta check
    Given a step

  Scenario: Gamma check
    Given a step
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "filter.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}, ScenarioRegex: "Alpha|Beta"}, nil
	}
	p := NewPlanner(cfgFn, nil)
	resp, err := p.Plan(context.Background(), &planv1.PlanRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws"},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(resp.Plan.Scenarios) != 2 {
		t.Errorf("expected 2 scenarios (Alpha, Beta), got %d", len(resp.Plan.Scenarios))
	}
	for _, sc := range resp.Plan.Scenarios {
		if sc.ScenarioName == "Gamma check" {
			t.Errorf("Gamma check should have been filtered out by scenario regex")
		}
	}
}

func TestPlan_scenarioRegexFilter_noMatch(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Filtering
  Scenario: Alpha check
    Given a step
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "filter.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}, ScenarioRegex: "Nonexistent"}, nil
	}
	p := NewPlanner(cfgFn, nil)
	resp, err := p.Plan(context.Background(), &planv1.PlanRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws"},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(resp.Plan.Scenarios) != 0 {
		t.Errorf("expected 0 scenarios with non-matching regex, got %d", len(resp.Plan.Scenarios))
	}
}

func TestPlan_scenarioRegexFilter_invalidRegex_returnsError(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Filtering
  Scenario: Alpha check
    Given a step
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "filter.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}, ScenarioRegex: `[invalid`}, nil
	}
	p := NewPlanner(cfgFn, nil)
	_, err := p.Plan(context.Background(), &planv1.PlanRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws"},
	})
	if err == nil {
		t.Fatal("expected error for invalid scenario regex")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("code: got %v want InvalidArgument", st.Code())
	}
}

func TestPlan_scenarioRegexAndTagFilter_combined(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Combined filters
  @smoke
  Scenario: Alpha smoke
    Given a step

  @smoke
  Scenario: Beta smoke
    Given a step

  Scenario: Gamma no-tag
    Given a step
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "combined.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		// ScenarioRegex filters to Alpha; TagExpression filters to @smoke.
		// Combined: only "Alpha smoke" should survive.
		return &RunConfig{FeaturePaths: []string{fp}, ScenarioRegex: "Alpha"}, nil
	}
	p := NewPlanner(cfgFn, nil)
	resp, err := p.Plan(context.Background(), &planv1.PlanRequest{
		Selector: &runv1.RunSelector{
			WorkspaceId:   "ws",
			TagExpression: "@smoke",
		},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(resp.Plan.Scenarios) != 1 {
		t.Errorf("expected 1 scenario (Alpha smoke), got %d", len(resp.Plan.Scenarios))
	}
	if len(resp.Plan.Scenarios) == 1 && resp.Plan.Scenarios[0].ScenarioName != "Alpha smoke" {
		t.Errorf("expected 'Alpha smoke', got %q", resp.Plan.Scenarios[0].ScenarioName)
	}
}

func TestPlan_scenarioHasMetadata(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Meta
  @tag-one @tag-two
  Scenario: Tagged scenario
    Given a step
`
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "meta.feature")
	if err := os.WriteFile(fp, []byte(featureContent), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}}, nil
	}
	p := NewPlanner(cfgFn, nil)
	resp, err := p.Plan(context.Background(), &planv1.PlanRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws"},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(resp.Plan.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(resp.Plan.Scenarios))
	}
	sc := resp.Plan.Scenarios[0]
	if sc.FeatureName != "Meta" {
		t.Errorf("FeatureName: got %q want %q", sc.FeatureName, "Meta")
	}
	if sc.ScenarioName != "Tagged scenario" {
		t.Errorf("ScenarioName: got %q want %q", sc.ScenarioName, "Tagged scenario")
	}
	if sc.ScenarioId == "" {
		t.Error("expected non-empty ScenarioId (DeterministicID)")
	}
	if len(sc.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(sc.Tags), sc.Tags)
	}
	if sc.DeterministicKey == nil {
		t.Error("expected non-nil DeterministicKey")
	}
}
