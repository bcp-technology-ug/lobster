package runner

import (
	"context"
	"fmt"
	"time"

	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	configv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/config"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	stackv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/stack"
	runstore "github.com/bcp-technology/lobster/gen/sqlc/run"
	"github.com/bcp-technology/lobster/internal/parser"
	"github.com/bcp-technology/lobster/internal/reports"
	"github.com/bcp-technology/lobster/internal/steps"
	"github.com/bcp-technology/lobster/internal/steps/builtin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RunSync parses features, executes matching scenarios serially, and streams
// RunEvent messages to the caller for real-time progress.
func (r *Runner) RunSync(ctx context.Context, req *runv1.RunSyncRequest, stream runv1.RunService_RunSyncServer) error {
	runID := newUUID()
	seq := newSeq()

	if err := stream.Send(runStatusEvent(runID, seq(), commonv1.RunStatus_RUN_STATUS_RUNNING, false)); err != nil {
		return err
	}

	cfg, err := r.resolveRunConfig(ctx, req.Selector, req.Execution)
	if err != nil {
		return err
	}

	if r.orchestrator != nil {
		if _, stackErr := r.orchestrator.EnsureStack(ctx, &stackv1.EnsureStackRequest{
			WorkspaceId:      req.Selector.WorkspaceId,
			ProfileName:      req.Selector.ProfileName,
			WaitForReadiness: true,
		}); stackErr != nil {
			return status.Errorf(codes.Internal, "ensure stack: %v", stackErr)
		}
	}

	features, err := r.loadFeatures(ctx, cfg, req.Selector)
	if err != nil {
		return err
	}

	reg := r.registry
	if reg == nil {
		reg = steps.NewRegistry()
		if regErr := builtin.Register(reg); regErr != nil {
			return status.Errorf(codes.Internal, "register builtin steps: %v", regErr)
		}
	}

	runResult := &reports.RunResult{
		RunID:     runID,
		Profile:   req.Selector.ProfileName,
		StartedAt: time.Now(),
	}
	reporter := reports.NewConsoleReporter(nil, true, false)
	reporter.RunStarted(runResult)

	// BeforeSuite hooks.
	if r.hooks != nil {
		if hookErr := r.hooks.RunBeforeSuite(ctx); hookErr != nil {
			return status.Errorf(codes.Internal, "before_suite hook: %v", hookErr)
		}
	}

	suiteVars := mergeMaps(cfg.Variables, req.Variables)

	for _, feature := range features {
		for _, scenario := range feature.Scenarios {
			if req.Selector.TagExpression != "" && !matchesTagExpression(scenario.Tags, req.Selector.TagExpression) {
				continue
			}
			sc := &reports.ScenarioResult{
				DeterministicID: scenario.DeterministicID,
				Name:            scenario.Name,
				FeatureName:     feature.Name,
				FeatureURI:      feature.URI,
				Tags:            scenario.Tags,
			}
			reporter.ScenarioStarted(sc)
			scenCtx := steps.NewScenarioContext(cfg.BaseURL, cfg.DefaultHeaders, suiteVars)
			scenCtx.SoftAssertMode = cfg.SoftAssert
			if r.hooks != nil {
				if hookErr := r.hooks.RunBeforeScenario(scenCtx); hookErr != nil {
					sc.Status = reports.StatusFailed
					sc.Err = hookErr
				} else {
					r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
					_ = r.hooks.RunAfterScenario(scenCtx)
				}
			} else {
				r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
			}
			reporter.ScenarioFinished(sc)
			runResult.Scenarios = append(runResult.Scenarios, sc)

			if err := stream.Send(scenarioResultEvent(runID, seq(), sc)); err != nil {
				return err
			}
			if cfg.FailFast && sc.Status == reports.StatusFailed {
				goto done
			}
		}
	}

done:
	// AfterSuite hooks (best-effort; error does not override run result).
	if r.hooks != nil {
		_ = r.hooks.RunAfterSuite(ctx)
	}
	runResult.Duration = time.Since(runResult.StartedAt)
	runResult.Finalize()
	reporter.RunFinished(runResult)

	if err := stream.Send(summaryEvent(runID, seq(), runResult)); err != nil {
		return err
	}
	termStatus := commonv1.RunStatus_RUN_STATUS_PASSED
	if runResult.Status == reports.StatusFailed {
		termStatus = commonv1.RunStatus_RUN_STATUS_FAILED
	}
	if err := stream.Send(runStatusEvent(runID, seq(), termStatus, true)); err != nil {
		return err
	}

	if r.orchestrator != nil && !cfg.KeepStack {
		_, _ = r.orchestrator.TeardownStack(ctx, &stackv1.TeardownStackRequest{
			WorkspaceId: req.Selector.WorkspaceId,
		})
	}
	return nil
}

// RunAsync persists a run record and starts execution in a background goroutine.
func (r *Runner) RunAsync(ctx context.Context, req *runv1.RunAsyncRequest) (*runv1.RunAsyncResponse, error) {
	runID := newUUID()
	now := time.Now().UTC()

	if r.store != nil {
		nowStr := now.Format(time.RFC3339Nano)
		var idempKey *string
		if req.IdempotencyKey != "" {
			s := req.IdempotencyKey
			idempKey = &s
		}
		var fpPtr, tagPtr, profPtr *string
		if req.Selector.FeaturePath != "" {
			s := req.Selector.FeaturePath
			fpPtr = &s
		}
		if req.Selector.TagExpression != "" {
			s := req.Selector.TagExpression
			tagPtr = &s
		}
		if req.Selector.ProfileName != "" {
			s := req.Selector.ProfileName
			profPtr = &s
		}
		if createErr := r.store.Run.CreateRun(ctx, runstore.CreateRunParams{
			RunID:                 runID,
			WorkspaceID:           req.Selector.WorkspaceId,
			ProfileName:           req.Selector.ProfileName,
			Status:                int64(commonv1.RunStatus_RUN_STATUS_PENDING),
			IdempotencyKey:        idempKey,
			SelectorFeaturePath:   fpPtr,
			SelectorTagExpression: tagPtr,
			SelectorProfileName:   profPtr,
			CreatedAt:             nowStr,
		}); createErr != nil {
			return nil, status.Errorf(codes.Internal, "create run record: %v", createErr)
		}
	}

	go r.executeAsync(runID, req)

	return &runv1.RunAsyncResponse{
		RunId:      runID,
		AcceptedAt: timestamppb.New(now),
	}, nil
}

// executeAsync runs scenarios in a background goroutine and persists outcome.
func (r *Runner) executeAsync(runID string, req *runv1.RunAsyncRequest) {
	ctx := context.Background()

	cfg, err := r.resolveRunConfig(ctx, req.Selector, req.Execution)
	if err != nil {
		r.updateRunStatus(ctx, runID, commonv1.RunStatus_RUN_STATUS_FAILED)
		return
	}
	r.updateRunStatus(ctx, runID, commonv1.RunStatus_RUN_STATUS_RUNNING)

	if r.orchestrator != nil {
		if _, sErr := r.orchestrator.EnsureStack(ctx, &stackv1.EnsureStackRequest{
			WorkspaceId:      req.Selector.WorkspaceId,
			ProfileName:      req.Selector.ProfileName,
			WaitForReadiness: true,
		}); sErr != nil {
			r.updateRunStatus(ctx, runID, commonv1.RunStatus_RUN_STATUS_FAILED)
			return
		}
	}

	features, err := r.loadFeatures(ctx, cfg, req.Selector)
	if err != nil {
		r.updateRunStatus(ctx, runID, commonv1.RunStatus_RUN_STATUS_FAILED)
		return
	}

	reg := r.registry
	if reg == nil {
		reg = steps.NewRegistry()
		_ = builtin.Register(reg)
	}

	runResult := &reports.RunResult{
		RunID:     runID,
		Profile:   req.Selector.ProfileName,
		StartedAt: time.Now(),
	}
	suiteVars := mergeMaps(cfg.Variables, req.Variables)
	reporter := reports.NewConsoleReporter(nil, false, false)

	if r.hooks != nil {
		if hookErr := r.hooks.RunBeforeSuite(ctx); hookErr != nil {
			r.updateRunStatus(ctx, runID, commonv1.RunStatus_RUN_STATUS_FAILED)
			return
		}
	}

	for _, feature := range features {
		for _, scenario := range feature.Scenarios {
			if req.Selector.TagExpression != "" && !matchesTagExpression(scenario.Tags, req.Selector.TagExpression) {
				continue
			}
			sc := &reports.ScenarioResult{
				DeterministicID: scenario.DeterministicID,
				Name:            scenario.Name,
				FeatureName:     feature.Name,
				FeatureURI:      feature.URI,
				Tags:            scenario.Tags,
			}
			scenCtx := steps.NewScenarioContext(cfg.BaseURL, cfg.DefaultHeaders, suiteVars)
			scenCtx.SoftAssertMode = cfg.SoftAssert
			if r.hooks != nil {
				if hookErr := r.hooks.RunBeforeScenario(scenCtx); hookErr != nil {
					sc.Status = reports.StatusFailed
					sc.Err = hookErr
				} else {
					r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
					_ = r.hooks.RunAfterScenario(scenCtx)
				}
			} else {
				r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
			}
			runResult.Scenarios = append(runResult.Scenarios, sc)
			if cfg.FailFast && sc.Status == reports.StatusFailed {
				goto asyncDone
			}
		}
	}

asyncDone:
	if r.hooks != nil {
		_ = r.hooks.RunAfterSuite(ctx)
	}
	runResult.Duration = time.Since(runResult.StartedAt)
	runResult.Finalize()

	finalStatus := commonv1.RunStatus_RUN_STATUS_PASSED
	if runResult.Status == reports.StatusFailed {
		finalStatus = commonv1.RunStatus_RUN_STATUS_FAILED
	}
	r.updateRunStatus(ctx, runID, finalStatus)
	r.updateRunSummary(ctx, runID, runResult)

	if r.orchestrator != nil && !cfg.KeepStack {
		_, _ = r.orchestrator.TeardownStack(ctx, &stackv1.TeardownStackRequest{
			WorkspaceId: req.Selector.WorkspaceId,
		})
	}
}

// executeScenario runs background + scenario steps and fills sc.
func (r *Runner) executeScenario(
	ctx context.Context,
	feature *parser.Feature,
	scenario *parser.Scenario,
	reg *steps.Registry,
	scenCtx *steps.ScenarioContext,
	sc *reports.ScenarioResult,
	cfg *RunConfig,
	reporter reports.Reporter,
) {
	start := time.Now()

	runStep := func(step *parser.Step) (stop bool) {
		sr := r.executeStep(ctx, step, reg, scenCtx, cfg)
		sc.Steps = append(sc.Steps, sr)
		reporter.StepFinished(sc, sr)
		if sr.Status == reports.StatusFailed && !cfg.SoftAssert {
			sc.Status = reports.StatusFailed
			return true
		}
		return false
	}

	if feature.Background != nil {
		for _, step := range feature.Background.Steps {
			if runStep(step) {
				sc.Duration = time.Since(start)
				return
			}
		}
	}
	for _, step := range scenario.Steps {
		if runStep(step) {
			sc.Duration = time.Since(start)
			return
		}
	}

	if cfg.SoftAssert && scenCtx.HasAssertionFailures() {
		sc.Status = reports.StatusFailed
		errs := scenCtx.AssertionErrors()
		msgs := make([]string, 0, len(errs))
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		sc.Err = fmt.Errorf("assertion failures: %v", msgs)
	} else {
		sc.Status = reports.StatusPassed
		for _, sr := range sc.Steps {
			if sr.Status == reports.StatusFailed {
				sc.Status = reports.StatusFailed
				break
			}
			if sr.Status == reports.StatusUndefined && sc.Status != reports.StatusFailed {
				sc.Status = reports.StatusUndefined
			}
		}
	}
	sc.Duration = time.Since(start)
}

// executeStep matches and runs a single step.
func (r *Runner) executeStep(
	ctx context.Context,
	step *parser.Step,
	reg *steps.Registry,
	scenCtx *steps.ScenarioContext,
	cfg *RunConfig,
) *reports.StepResult {
	sr := &reports.StepResult{Keyword: step.Keyword, Text: step.Text}

	stepDef, args, matchErr := reg.MatchStep(step)
	if matchErr != nil {
		sr.Status = reports.StatusUndefined
		sr.Err = matchErr
		return sr
	}

	scenCtx.CurrentStep = step
	execCtx := ctx
	if cfg.StepTimeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, cfg.StepTimeout)
		defer cancel()
	}

	start := time.Now()
	handlerErr := stepDef.Handler(scenCtx, args...)
	_ = execCtx
	sr.Duration = time.Since(start)

	if handlerErr != nil {
		sr.Status = reports.StatusFailed
		sr.Err = handlerErr
	} else {
		sr.Status = reports.StatusPassed
	}
	return sr
}

// --- helpers ---

func (r *Runner) resolveRunConfig(ctx context.Context, sel *runv1.RunSelector, exec *configv1.ExecutionConfig) (*RunConfig, error) {
	if sel == nil {
		return nil, status.Error(codes.InvalidArgument, "selector is required")
	}
	cfg, err := r.configFn(ctx, sel.WorkspaceId, sel.ProfileName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "resolve config: %v", err)
	}
	if exec != nil {
		if exec.SoftAssert {
			cfg.SoftAssert = true
		}
		if exec.FailFast {
			cfg.FailFast = true
		}
		if exec.KeepStack {
			cfg.KeepStack = true
		}
		if exec.StepTimeout != nil {
			cfg.StepTimeout = exec.StepTimeout.AsDuration()
		}
		if exec.RunTimeout != nil {
			cfg.RunTimeout = exec.RunTimeout.AsDuration()
		}
	}
	return cfg, nil
}

func (r *Runner) loadFeatures(ctx context.Context, cfg *RunConfig, sel *runv1.RunSelector) ([]*parser.Feature, error) {
	_ = ctx
	paths := cfg.FeaturePaths
	if sel != nil && sel.FeaturePath != "" {
		paths = []string{sel.FeaturePath}
	}
	var features []*parser.Feature
	for _, fp := range paths {
		parsed, err := parser.ParseGlob(fp)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "parse %q: %v", fp, err)
		}
		features = append(features, parsed...)
	}
	return features, nil
}

func (r *Runner) updateRunStatus(ctx context.Context, runID string, s commonv1.RunStatus) {
	if r.store == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var startedAt, endedAt *string
	if s == commonv1.RunStatus_RUN_STATUS_RUNNING {
		startedAt = &now
	}
	if s == commonv1.RunStatus_RUN_STATUS_PASSED || s == commonv1.RunStatus_RUN_STATUS_FAILED || s == commonv1.RunStatus_RUN_STATUS_CANCELLED {
		endedAt = &now
	}
	_ = r.store.Run.UpdateRunStatus(ctx, runstore.UpdateRunStatusParams{
		RunID:     runID,
		Status:    int64(s),
		StartedAt: startedAt,
		EndedAt:   endedAt,
	})
}

func (r *Runner) updateRunSummary(ctx context.Context, runID string, res *reports.RunResult) {
	if r.store == nil {
		return
	}
	_ = r.store.Run.UpdateRunSummary(ctx, runstore.UpdateRunSummaryParams{
		RunID:                   runID,
		SummaryTotalScenarios:   int64(res.Total),
		SummaryPassedScenarios:  int64(res.Passed),
		SummaryFailedScenarios:  int64(res.Failed),
		SummarySkippedScenarios: int64(res.Skipped),
		SummaryDurationNanos:    int64(res.Duration),
	})
}

func newSeq() func() uint64 {
	var n uint64
	return func() uint64 { n++; return n }
}

func mergeMaps(base, override map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

// --- RunEvent constructors ---

func runStatusEvent(runID string, seq uint64, s commonv1.RunStatus, terminal bool) *runv1.RunEvent {
	return &runv1.RunEvent{
		Sequence:   seq,
		RunId:      runID,
		ObservedAt: timestamppb.Now(),
		EventType:  runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS,
		Terminal:   terminal,
		Payload:    &runv1.RunEvent_RunStatus{RunStatus: s},
	}
}

func scenarioResultEvent(runID string, seq uint64, sc *reports.ScenarioResult) *runv1.RunEvent {
	sr := &runv1.ScenarioResult{
		ScenarioId: sc.DeterministicID,
		Status:     reportsStatusToScenarioProto(sc.Status),
		Duration:   durationpb.New(sc.Duration),
	}
	for _, step := range sc.Steps {
		sr.StepResults = append(sr.StepResults, &runv1.StepResult{
			Status:   stepStatusToProto(step.Status),
			Duration: durationpb.New(step.Duration),
		})
	}
	return &runv1.RunEvent{
		Sequence:   seq,
		RunId:      runID,
		ObservedAt: timestamppb.Now(),
		EventType:  runv1.RunEventType_RUN_EVENT_TYPE_SCENARIO_RESULT,
		Payload:    &runv1.RunEvent_ScenarioResult{ScenarioResult: sr},
	}
}

func summaryEvent(runID string, seq uint64, res *reports.RunResult) *runv1.RunEvent {
	return &runv1.RunEvent{
		Sequence:   seq,
		RunId:      runID,
		ObservedAt: timestamppb.Now(),
		EventType:  runv1.RunEventType_RUN_EVENT_TYPE_SUMMARY,
		Payload: &runv1.RunEvent_Summary{Summary: &runv1.RunSummary{
			TotalScenarios:   uint32(res.Total),
			PassedScenarios:  uint32(res.Passed),
			FailedScenarios:  uint32(res.Failed),
			SkippedScenarios: uint32(res.Skipped),
			Duration:         durationpb.New(res.Duration),
		}},
	}
}

func reportsStatusToScenarioProto(s reports.Status) commonv1.ScenarioStatus {
	switch s {
	case reports.StatusPassed:
		return commonv1.ScenarioStatus_SCENARIO_STATUS_PASSED
	case reports.StatusFailed:
		return commonv1.ScenarioStatus_SCENARIO_STATUS_FAILED
	case reports.StatusSkipped:
		return commonv1.ScenarioStatus_SCENARIO_STATUS_SKIPPED
	default:
		return commonv1.ScenarioStatus_SCENARIO_STATUS_UNSPECIFIED
	}
}

func stepStatusToProto(s reports.Status) commonv1.StepStatus {
	switch s {
	case reports.StatusPassed:
		return commonv1.StepStatus_STEP_STATUS_PASSED
	case reports.StatusFailed:
		return commonv1.StepStatus_STEP_STATUS_FAILED
	case reports.StatusUndefined:
		return commonv1.StepStatus_STEP_STATUS_UNDEFINED
	case reports.StatusSkipped:
		return commonv1.StepStatus_STEP_STATUS_SKIPPED
	default:
		return commonv1.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}
