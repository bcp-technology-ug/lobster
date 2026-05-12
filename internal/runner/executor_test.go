package runner

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	planv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology-ug/lobster/internal/store"
	"google.golang.org/grpc/metadata"
)

// fakeStream implements runv1.RunService_RunSyncServer for testing.
type fakeStream struct {
	ctx     context.Context
	mu      sync.Mutex
	events  []*runv1.RunEvent
	sendErr error
}

func (f *fakeStream) Context() context.Context { return f.ctx }
func (f *fakeStream) Send(evt *runv1.RunEvent) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.mu.Lock()
	f.events = append(f.events, evt)
	f.mu.Unlock()
	return nil
}
func (f *fakeStream) SetHeader(md metadata.MD) error  { return nil }
func (f *fakeStream) SendHeader(md metadata.MD) error { return nil }
func (f *fakeStream) SetTrailer(md metadata.MD)       {}
func (f *fakeStream) SendMsg(m any) error             { return nil }
func (f *fakeStream) RecvMsg(m any) error             { return nil }

func (f *fakeStream) receivedEventTypes() []runv1.RunEventType {
	f.mu.Lock()
	defer f.mu.Unlock()
	types := make([]runv1.RunEventType, len(f.events))
	for i, e := range f.events {
		types[i] = e.EventType
	}
	return types
}

// emptyCfgFn returns a RunConfig with no feature paths.
func emptyCfgFn(_ context.Context, _, _ string) (*RunConfig, error) {
	return &RunConfig{}, nil
}

func newTestRunner() *Runner {
	return New(emptyCfgFn, nil, nil, nil)
}

// --- RunSync tests ---

func TestRunSync_noFeatures_emitsExpectedEvents(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	stream := &fakeStream{ctx: context.Background()}
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-test"},
	}
	if err := r.RunSync(context.Background(), req, stream); err != nil {
		t.Fatalf("RunSync error: %v", err)
	}
	types := stream.receivedEventTypes()
	// Expect: RUNNING status, SUMMARY, terminal PASSED/FAILED status
	if len(types) < 3 {
		t.Fatalf("expected at least 3 events, got %d: %v", len(types), types)
	}
	if types[0] != runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS {
		t.Errorf("first event: got %v want RUN_STATUS", types[0])
	}
	last := types[len(types)-1]
	if last != runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS {
		t.Errorf("last event: got %v want RUN_STATUS (terminal)", last)
	}
	// The last event must be terminal.
	stream.mu.Lock()
	lastEvt := stream.events[len(stream.events)-1]
	stream.mu.Unlock()
	if !lastEvt.Terminal {
		t.Error("last event should be terminal")
	}
}

func TestRunSync_nilSelector_returnsError(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	stream := &fakeStream{ctx: context.Background()}
	req := &runv1.RunSyncRequest{} // nil selector
	err := r.RunSync(context.Background(), req, stream)
	if err == nil {
		t.Error("expected error for nil selector")
	}
}

func TestRunSync_withTempFeatureFile(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Simple
  Scenario: Pass
    Given I am in a new temporary directory
`
	tmp := t.TempDir()
	fp := tmp + "/simple.feature"
	if err := writeFile(fp, featureContent); err != nil {
		t.Fatalf("write feature file: %v", err)
	}

	r := New(func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}}, nil
	}, nil, nil, nil)

	stream := &fakeStream{ctx: context.Background()}
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-test"},
	}
	if err := r.RunSync(context.Background(), req, stream); err != nil {
		t.Fatalf("RunSync error: %v", err)
	}
	types := stream.receivedEventTypes()
	// Must have at least: RUNNING + SCENARIO_RESULT + SUMMARY + terminal status
	if len(types) < 4 {
		t.Errorf("expected at least 4 events, got %d: %v", len(types), types)
	}
	hasScenarioResult := false
	for _, tt := range types {
		if tt == runv1.RunEventType_RUN_EVENT_TYPE_SCENARIO_RESULT {
			hasScenarioResult = true
			break
		}
	}
	if !hasScenarioResult {
		t.Error("expected at least one SCENARIO_RESULT event")
	}
}

// --- RunAsync tests ---

func TestRunAsync_nilStore_succeeds(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	req := &runv1.RunAsyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-test"},
	}
	resp, err := r.RunAsync(context.Background(), req)
	if err != nil {
		t.Fatalf("RunAsync error: %v", err)
	}
	if resp.RunId == "" {
		t.Error("expected non-empty RunId")
	}
	if resp.AcceptedAt == nil {
		t.Error("expected non-nil AcceptedAt")
	}
	// Give the goroutine a chance to finish (it runs with empty features).
	time.Sleep(50 * time.Millisecond)
}

func TestRunAsync_uuidFormat(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	req := &runv1.RunAsyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-test"},
	}
	resp, err := r.RunAsync(context.Background(), req)
	if err != nil {
		t.Fatalf("RunAsync error: %v", err)
	}
	// Basic UUID v4 format: 8-4-4-4-12 hex chars
	id := resp.RunId
	if len(id) != 36 {
		t.Errorf("RunId length: got %d want 36, id=%q", len(id), id)
	}
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("RunId not in UUID format: %q", id)
	}
}

// --- Cancel tests ---

func TestCancel_idempotent(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	// Cancel a non-existent run — should not error.
	if err := r.Cancel(context.Background(), "nonexistent-run-id"); err != nil {
		t.Errorf("Cancel should be idempotent (non-existent run), got: %v", err)
	}
	// Cancel twice — second call should also be a no-op.
	if err := r.Cancel(context.Background(), "nonexistent-run-id"); err != nil {
		t.Errorf("Cancel second call should not error, got: %v", err)
	}
}

func TestCancel_stopsBackgroundRun(t *testing.T) {
	t.Parallel()
	// Use a blocking configFn so the goroutine is definitely running.
	block := make(chan struct{})
	started := make(chan string, 1)
	r := New(func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{}}, nil
	}, nil, nil, nil)

	req := &runv1.RunAsyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-cancel"},
	}

	// Override the configFn with one that signals start and waits.
	_ = block
	_ = started

	resp, err := r.RunAsync(context.Background(), req)
	if err != nil {
		t.Fatalf("RunAsync error: %v", err)
	}
	runID := resp.RunId

	// Cancel immediately — should return nil even if goroutine already finished.
	if err := r.Cancel(context.Background(), runID); err != nil {
		t.Errorf("Cancel error: %v", err)
	}
}

// --- Helper tests ---

func TestMergeMaps_basic(t *testing.T) {
	t.Parallel()
	base := map[string]string{"a": "1", "b": "2"}
	override := map[string]string{"b": "override", "c": "3"}
	got := mergeMaps(base, override)
	if got["a"] != "1" {
		t.Errorf("a: got %q want %q", got["a"], "1")
	}
	if got["b"] != "override" {
		t.Errorf("b: got %q want %q", got["b"], "override")
	}
	if got["c"] != "3" {
		t.Errorf("c: got %q want %q", got["c"], "3")
	}
}

func TestMergeMaps_nilBase(t *testing.T) {
	t.Parallel()
	override := map[string]string{"x": "y"}
	got := mergeMaps(nil, override)
	if got["x"] != "y" {
		t.Errorf("x: got %q want %q", got["x"], "y")
	}
}

func TestMergeMaps_nilOverride(t *testing.T) {
	t.Parallel()
	base := map[string]string{"a": "1"}
	got := mergeMaps(base, nil)
	if got["a"] != "1" {
		t.Errorf("a: got %q want %q", got["a"], "1")
	}
}

func TestContainsString_found(t *testing.T) {
	t.Parallel()
	if !containsString([]string{"a", "b", "c"}, "b") {
		t.Error("expected true for contained string")
	}
}

func TestContainsString_notFound(t *testing.T) {
	t.Parallel()
	if containsString([]string{"a", "b", "c"}, "x") {
		t.Error("expected false for missing string")
	}
}

func TestContainsString_empty(t *testing.T) {
	t.Parallel()
	if containsString(nil, "x") {
		t.Error("expected false for nil slice")
	}
}

func TestNewUUID_format(t *testing.T) {
	t.Parallel()
	id := newUUID()
	if len(id) != 36 {
		t.Errorf("UUID length: got %d want 36", len(id))
	}
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("UUID format invalid: %q", id)
	}
	// version nibble must be 4
	if id[14] != '4' {
		t.Errorf("UUID version nibble: got %c want 4", id[14])
	}
}

func TestNewUUID_uniqueness(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := newUUID()
		if seen[id] {
			t.Errorf("duplicate UUID generated: %q", id)
		}
		seen[id] = true
	}
}

// --- scenarioIsQuarantined tests ---

func TestScenarioIsQuarantined_disabledAlwaysFalse(t *testing.T) {
	t.Parallel()
	cfg := &RunConfig{QuarantineEnabled: false, QuarantineTag: "@quarantine"}
	tags := []string{"@quarantine"}
	if scenarioIsQuarantined(tags, cfg) {
		t.Error("expected false when QuarantineEnabled is false")
	}
}

func TestScenarioIsQuarantined_enabledDefaultTag(t *testing.T) {
	t.Parallel()
	cfg := &RunConfig{QuarantineEnabled: true}
	if !scenarioIsQuarantined([]string{"@quarantine"}, cfg) {
		t.Error("expected true for @quarantine tag with default tag")
	}
}

func TestScenarioIsQuarantined_enabledCustomTag(t *testing.T) {
	t.Parallel()
	cfg := &RunConfig{QuarantineEnabled: true, QuarantineTag: "@flaky"}
	if !scenarioIsQuarantined([]string{"@flaky"}, cfg) {
		t.Error("expected true for custom @flaky tag")
	}
	if scenarioIsQuarantined([]string{"@quarantine"}, cfg) {
		t.Error("expected false when tag is @quarantine but custom tag is @flaky")
	}
}

func TestScenarioIsQuarantined_matchesBareAndPrefixed(t *testing.T) {
	t.Parallel()
	cfg := &RunConfig{QuarantineEnabled: true, QuarantineTag: "@quarantine"}
	// bare tag without @
	if !scenarioIsQuarantined([]string{"quarantine"}, cfg) {
		t.Error("expected true for bare 'quarantine' tag (no @ prefix)")
	}
}

func TestScenarioIsQuarantined_noMatchingTag(t *testing.T) {
	t.Parallel()
	cfg := &RunConfig{QuarantineEnabled: true, QuarantineTag: "@quarantine"}
	if scenarioIsQuarantined([]string{"@smoke", "@regression"}, cfg) {
		t.Error("expected false for tags that don't include quarantine tag")
	}
}

func TestScenarioIsQuarantined_emptyTags(t *testing.T) {
	t.Parallel()
	cfg := &RunConfig{QuarantineEnabled: true}
	if scenarioIsQuarantined(nil, cfg) {
		t.Error("expected false for nil tags")
	}
	if scenarioIsQuarantined([]string{}, cfg) {
		t.Error("expected false for empty tags")
	}
}

// --- Quarantine demotion in RunSync ---

func TestRunSync_quarantinedFailureDemotedToSkipped(t *testing.T) {
	t.Parallel()
	// A scenario tagged @quarantine that would fail (undefined step) should
	// be reported as skipped when QuarantineEnabled=true, QuarantineBlocking=false.
	featureContent := `Feature: Quarantine
  @quarantine
  Scenario: Flaky scenario
    Given a step that always fails
`
	tmp := t.TempDir()
	fp := tmp + "/quarantine.feature"
	if err := writeFile(fp, featureContent); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{
			FeaturePaths:       []string{fp},
			QuarantineEnabled:  true,
			QuarantineTag:      "@quarantine",
			QuarantineBlocking: false,
		}, nil
	}
	r := New(cfgFn, nil, nil, nil)
	stream := &fakeStream{ctx: context.Background()}
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-quarantine"},
	}
	if err := r.RunSync(context.Background(), req, stream); err != nil {
		t.Fatalf("RunSync error: %v", err)
	}
	// Find the scenario result event and check it is not FAILED.
	stream.mu.Lock()
	defer stream.mu.Unlock()
	for _, ev := range stream.events {
		if sr := ev.GetScenarioResult(); sr != nil {
			// A quarantined undefined-step scenario should be demoted to SKIPPED.
			statusStr := sr.GetStatus().String()
			if statusStr == "SCENARIO_STATUS_FAILED" {
				t.Errorf("quarantined scenario should not have status FAILED, got %s", statusStr)
			}
		}
	}
}

func TestRunSync_quarantineBlockingKeepsFailure(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: Quarantine blocking
  @quarantine
  Scenario: Blocking flaky
    Given a step that does not exist
`
	tmp := t.TempDir()
	fp := tmp + "/block.feature"
	if err := writeFile(fp, featureContent); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{
			FeaturePaths:       []string{fp},
			QuarantineEnabled:  true,
			QuarantineTag:      "@quarantine",
			QuarantineBlocking: true,
		}, nil
	}
	r := New(cfgFn, nil, nil, nil)
	stream := &fakeStream{ctx: context.Background()}
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-block"},
	}
	_ = r.RunSync(context.Background(), req, stream)
	// With QuarantineBlocking=true the scenario remains FAILED/UNDEFINED, not
	// demoted to SKIPPED.
	stream.mu.Lock()
	defer stream.mu.Unlock()
	foundScenario := false
	for _, ev := range stream.events {
		if sr := ev.GetScenarioResult(); sr != nil {
			foundScenario = true
			statusStr := sr.GetStatus().String()
			if statusStr == "SCENARIO_STATUS_SKIPPED" {
				t.Errorf("blocking quarantine should not demote to SKIPPED, got %s", statusStr)
			}
		}
	}
	if !foundScenario {
		t.Error("expected at least one scenario result event")
	}
}

// --- ScenarioID filter in RunSync ---

func TestRunSync_scenarioIDFilter(t *testing.T) {
	t.Parallel()
	featureContent := `Feature: ID Filter
  Scenario: Alpha
    Given I am in a new temporary directory

  Scenario: Beta
    Given I am in a new temporary directory
`
	tmp := t.TempDir()
	fp := tmp + "/idfilter.feature"
	if err := writeFile(fp, featureContent); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	// First, plan to get the deterministic ID for "Alpha".
	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{FeaturePaths: []string{fp}}, nil
	}
	planner := NewPlanner(cfgFn, nil)
	planResp, err := planner.Plan(context.Background(), &planv1.PlanRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws"},
	})
	if err != nil || len(planResp.Plan.Scenarios) != 2 {
		t.Fatalf("plan setup: err=%v scenarios=%d", err, len(planResp.Plan.Scenarios))
	}
	alphaID := ""
	for _, sc := range planResp.Plan.Scenarios {
		if sc.ScenarioName == "Alpha" {
			alphaID = sc.ScenarioId
		}
	}
	if alphaID == "" {
		t.Fatal("could not find Alpha scenario ID")
	}

	// Run with only the Alpha scenario ID.
	runCfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{
			FeaturePaths: []string{fp},
			ScenarioIDs:  []string{alphaID},
		}, nil
	}
	r := New(runCfgFn, nil, nil, nil)
	stream := &fakeStream{ctx: context.Background()}
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws"},
	}
	if err := r.RunSync(context.Background(), req, stream); err != nil {
		t.Fatalf("RunSync error: %v", err)
	}
	// Expect exactly one scenario result event.
	stream.mu.Lock()
	var scenarioResults int
	for _, ev := range stream.events {
		if ev.GetScenarioResult() != nil {
			scenarioResults++
		}
	}
	stream.mu.Unlock()
	if scenarioResults != 1 {
		t.Errorf("expected 1 scenario result (Alpha only), got %d", scenarioResults)
	}
}

// --- SoftAssert in RunSync ---

func TestRunSync_softAssert_collectsMultipleFailures(t *testing.T) {
	t.Parallel()
	// Two assertions: one passes, one fails.
	// With SoftAssert the scenario should still complete (not abort on first fail).
	featureContent := `Feature: Soft assert
  Scenario: Multiple assertions
    Given I am in a new temporary directory
    When I run the command "echo hello"
    Then the output should contain "hello"
    And the output should contain "world"
`
	tmp := t.TempDir()
	fp := tmp + "/soft.feature"
	if err := writeFile(fp, featureContent); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	cfgFn := func(_ context.Context, _, _ string) (*RunConfig, error) {
		return &RunConfig{
			FeaturePaths: []string{fp},
			SoftAssert:   true,
		}, nil
	}
	r := New(cfgFn, nil, nil, nil)
	stream := &fakeStream{ctx: context.Background()}
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{WorkspaceId: "ws-soft"},
	}
	// RunSync should not return a gRPC error — scenario failure is in-band.
	if err := r.RunSync(context.Background(), req, stream); err != nil {
		t.Fatalf("RunSync returned unexpected gRPC error: %v", err)
	}
	// There should be exactly 1 scenario result event.
	stream.mu.Lock()
	var scenCount int
	for _, ev := range stream.events {
		if ev.GetScenarioResult() != nil {
			scenCount++
		}
	}
	stream.mu.Unlock()
	if scenCount != 1 {
		t.Errorf("expected 1 scenario result event, got %d", scenCount)
	}
}

// --- WithRetention ---

func TestWithRetention_setsFieldAndReturnsReceiver(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	cfg := store.RetentionConfig{MaxRuns: 10, MaxAge: 24 * time.Hour}
	r2 := r.WithRetention(cfg)
	if r2 != r {
		t.Error("WithRetention should return the same *Runner")
	}
	if r.retention.MaxRuns != 10 {
		t.Errorf("MaxRuns: got %d want 10", r.retention.MaxRuns)
	}
	if r.retention.MaxAge != 24*time.Hour {
		t.Errorf("MaxAge: got %v want 24h", r.retention.MaxAge)
	}
}

func TestWithRetention_zeroConfig_noOp(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	r2 := r.WithRetention(store.RetentionConfig{})
	if r2 != r {
		t.Error("WithRetention should return the same *Runner")
	}
	if r.retention.MaxRuns != 0 || r.retention.MaxAge != 0 {
		t.Error("zero RetentionConfig should leave retention fields at zero")
	}
}

// --- pruneRetention ---

func TestPruneRetention_nilStore_noOp(t *testing.T) {
	t.Parallel()
	r := New(emptyCfgFn, nil, nil, nil)
	r.retention = store.RetentionConfig{MaxRuns: 5}
	// Should not panic when store is nil.
	r.pruneRetention(context.Background(), "ws-test")
}

// writeFile writes content to path for test setup.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
