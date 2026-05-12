package runner

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	configv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/config"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
	"github.com/bcp-technology-ug/lobster/internal/log"
	"github.com/bcp-technology-ug/lobster/internal/parser"
	"github.com/bcp-technology-ug/lobster/internal/reports"
	"github.com/bcp-technology-ug/lobster/internal/steps"
	"github.com/bcp-technology-ug/lobster/internal/steps/builtin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RunSync parses features, executes matching scenarios serially, and streams
// RunEvent messages to the caller for real-time progress. It also persists
// the run record and events to the store when a store is configured, enabling
// GetRun / StreamRunEvents replay for sync runs.
func (r *Runner) RunSync(ctx context.Context, req *runv1.RunSyncRequest, stream runv1.RunService_RunSyncServer) error {
	// Start a root OTel span for the entire run. When OTel is not configured
	// the global provider is a no-op and this is a zero-cost call.
	tracer := otel.GetTracerProvider().Tracer("lobster/runner")
	ctx, runSpan := tracer.Start(ctx, "lobster.run")
	defer runSpan.End()
	if req.Selector != nil {
		runSpan.SetAttributes(
			attribute.String("lobster.workspace_id", req.Selector.WorkspaceId),
			attribute.String("lobster.profile", req.Selector.ProfileName),
		)
	}
	runID := newUUID()
	seq := newSeq()
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	// Open the in-process event bus before persisting or sending any events so
	// that a concurrent StreamRunEvents call subscribes before we publish.
	r.openBus(runID)
	defer r.closeBus(runID)

	// Persist run record so it is visible via GetRun / ListRuns immediately.
	if r.store != nil {
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
			Status:                int64(commonv1.RunStatus_RUN_STATUS_RUNNING),
			SelectorFeaturePath:   fpPtr,
			SelectorTagExpression: tagPtr,
			SelectorProfileName:   profPtr,
			CreatedAt:             nowStr,
		}); createErr != nil {
			return status.Errorf(grpccodes.Internal, "create run record: %v", createErr)
		}
		if updateErr := r.store.Run.UpdateRunStatus(ctx, runstore.UpdateRunStatusParams{
			RunID:     runID,
			Status:    int64(commonv1.RunStatus_RUN_STATUS_RUNNING),
			StartedAt: &nowStr,
		}); updateErr != nil {
			log.FromContext(ctx).Warn("failed to update run status to RUNNING", zap.String("run_id", runID), zap.Error(updateErr))
		}
	}

	initialEvt := runStatusEvent(runID, seq(), commonv1.RunStatus_RUN_STATUS_RUNNING, false)
	r.appendEvent(ctx, runID, initialEvt)
	if err := stream.Send(initialEvt); err != nil {
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
			return status.Errorf(grpccodes.Internal, "ensure stack: %v", stackErr)
		}
	}

	// Adapter setup runs once before the suite.
	if r.adapters != nil {
		if setupErr := r.adapters.SetupAll(ctx); setupErr != nil {
			return status.Errorf(grpccodes.Internal, "adapter setup: %v", setupErr)
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
			return status.Errorf(grpccodes.Internal, "register builtin steps: %v", regErr)
		}
	}
	if r.hooks == nil {
		h := steps.NewHookRegistry()
		builtin.RegisterHooks(h)
		r.hooks = h
	}

	runResult := &reports.RunResult{
		RunID:     runID,
		Profile:   req.Selector.ProfileName,
		StartedAt: time.Now(),
	}
	reporter := r.reporter
	if reporter == nil {
		reporter = reports.NewConsoleReporter(nil, true, false)
	}
	reporter.RunStarted(runResult)

	// BeforeSuite hooks.
	if r.hooks != nil {
		if hookErr := r.hooks.RunBeforeSuite(ctx); hookErr != nil {
			return status.Errorf(grpccodes.Internal, "before_suite hook: %v", hookErr)
		}
	}

	suiteVars := mergeMaps(cfg.Variables, req.Variables)

	// scenarioRegex is pre-compiled in resolveRunConfig; use directly.
	scenarioRegex := cfg.CompiledScenarioRegex

	// scenarioOrdinal tracks insertion order for the run_scenarios table.
	var scenarioOrdinal int64

	for _, feature := range features {
		for _, scenario := range feature.Scenarios {
			if req.Selector.TagExpression != "" && !matchesTagExpression(scenario.Tags, req.Selector.TagExpression) {
				continue
			}
			if scenarioRegex != nil && !scenarioRegex.MatchString(scenario.Name) {
				continue
			}
			if len(cfg.ScenarioIDs) > 0 && !containsString(cfg.ScenarioIDs, scenario.DeterministicID) {
				continue
			}
			sc := &reports.ScenarioResult{
				DeterministicID: scenario.DeterministicID,
				Name:            scenario.Name,
				FeatureName:     feature.Name,
				FeatureURI:      feature.URI,
				Tags:            scenario.Tags,
			}
			// Per-scenario OTel span.
			_, scenSpan := tracer.Start(ctx, "lobster.scenario") // Use start options for scenario attributes

			scenSpan.SetAttributes(
				attribute.String("lobster.scenario.name", scenario.Name),
				attribute.String("lobster.feature", feature.Name),
			)
			reporter.ScenarioStarted(sc)
			scenCtx := steps.NewScenarioContext(cfg.BaseURL, cfg.DefaultHeaders, suiteVars)
			scenCtx.SoftAssertMode = cfg.SoftAssert

			// Adapter reset runs before each scenario for clean state.
			if r.adapters != nil {
				if resetErr := r.adapters.ResetAll(ctx); resetErr != nil {
					sc.Status = reports.StatusFailed
					sc.Err = resetErr
				}
			}

			if sc.Status != reports.StatusFailed {
				if r.hooks != nil {
					if hookErr := r.hooks.RunBeforeScenario(scenCtx); hookErr != nil {
						sc.Status = reports.StatusFailed
						sc.Err = hookErr
					} else {
						r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
						if afterErr := r.hooks.RunAfterScenario(scenCtx); afterErr != nil {
							log.FromContext(ctx).Warn("AfterScenario hook error", zap.String("run_id", runID), zap.Error(afterErr))
						}
					}
				} else {
					r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
				}
			}
			// Quarantine: when a scenario fails but is non-blocking quarantined,
			// demote its status to Skipped so the overall run result is unaffected.
			if sc.Status == reports.StatusFailed && !cfg.QuarantineBlocking && scenarioIsQuarantined(scenario.Tags, cfg) {
				sc.Status = reports.StatusSkipped
			}
			// Finish the scenario span with error status if the scenario failed.
			if sc.Status == reports.StatusFailed {
				scenSpan.SetStatus(codes.Error, "scenario failed")
			}
			scenSpan.End()
			reporter.ScenarioFinished(sc)
			runResult.Scenarios = append(runResult.Scenarios, sc)

			// run_events FK constraint on (run_id, payload_scenario_id) is
			// satisfied. This is required both in daemon mode (where the API
			// service may not have pre-populated scenario rows) and in local
			// CLI mode (where no service layer exists at all).
			scenarioOrdinal++
			if r.store != nil {
				featName := feature.Name
				if upsertErr := r.store.Run.UpsertRunScenario(ctx, runstore.UpsertRunScenarioParams{
					RunID:         runID,
					ScenarioID:    sc.DeterministicID,
					Ordinal:       scenarioOrdinal,
					Name:          sc.Name,
					Status:        int64(reportsStatusToScenarioProto(sc.Status)),
					DurationNanos: sc.Duration.Nanoseconds(),
					FeatureName:   &featName,
				}); upsertErr != nil {
					log.FromContext(ctx).Warn("failed to persist scenario record",
						zap.String("run_id", runID),
						zap.String("scenario_id", sc.DeterministicID),
						zap.Error(upsertErr))
				}
				r.persistStepDetail(ctx, runID, scenario, sc)
			}

			scEvt := scenarioResultEvent(runID, seq(), sc)
			r.appendEvent(ctx, runID, scEvt)
			if err := stream.Send(scEvt); err != nil {
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
		if afterSuiteErr := r.hooks.RunAfterSuite(ctx); afterSuiteErr != nil {
			log.FromContext(ctx).Warn("AfterSuite hook error", zap.String("run_id", runID), zap.Error(afterSuiteErr))
		}
	}
	// Adapter teardown after suite (best-effort).
	if r.adapters != nil {
		if teardownErr := r.adapters.TeardownAll(ctx); teardownErr != nil {
			log.FromContext(ctx).Warn("adapter teardown error", zap.String("run_id", runID), zap.Error(teardownErr))
		}
	}
	runResult.Duration = time.Since(runResult.StartedAt)
	runResult.Finalize()
	reporter.RunFinished(runResult)

	sumEvt := summaryEvent(runID, seq(), runResult)
	r.appendEvent(ctx, runID, sumEvt)
	if err := stream.Send(sumEvt); err != nil {
		return err
	}
	termStatus := commonv1.RunStatus_RUN_STATUS_PASSED
	if runResult.Status == reports.StatusFailed {
		termStatus = commonv1.RunStatus_RUN_STATUS_FAILED
	}
	termEvt := runStatusEvent(runID, seq(), termStatus, true)
	r.appendEvent(ctx, runID, termEvt)
	if err := stream.Send(termEvt); err != nil {
		return err
	}

	r.updateRunStatus(ctx, runID, termStatus)
	r.updateRunSummary(ctx, runID, runResult)

	if r.orchestrator != nil && !cfg.KeepStack {
		if _, teardownErr := r.orchestrator.TeardownStack(ctx, &stackv1.TeardownStackRequest{
			WorkspaceId: req.Selector.WorkspaceId,
		}); teardownErr != nil {
			log.FromContext(ctx).Warn("stack teardown error", zap.String("run_id", runID), zap.Error(teardownErr))
		}
	}
	// Best-effort retention pruning after run completes.
	r.pruneRetention(ctx, req.Selector.WorkspaceId)
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
			return nil, status.Errorf(grpccodes.Internal, "create run record: %v", createErr)
		}
	}

	// Open the in-process event bus before spawning so that StreamRunEvents can
	// subscribe and receive all events including the first status update.
	r.openBus(runID)

	// Register a cancellable context so CancelRun can interrupt this goroutine.
	cancelCtx, cancel := context.WithCancel(context.Background())
	r.cancelMu.Lock()
	r.cancels[runID] = cancel
	r.cancelMu.Unlock()

	// Carry the logger from the request context into the background goroutine.
	bgLogger := log.FromContext(ctx).With(zap.String("run_id", runID))
	bgCtx := log.WithLogger(cancelCtx, bgLogger)

	go func() {
		defer func() {
			r.cancelMu.Lock()
			delete(r.cancels, runID)
			r.cancelMu.Unlock()
			cancel()
			r.closeBus(runID)
		}()
		// Acquire semaphore slot (blocks if at capacity).
		if r.sem != nil {
			r.sem <- struct{}{}
			defer func() { <-r.sem }()
		}
		r.executeAsync(runID, req, bgCtx)
	}()

	return &runv1.RunAsyncResponse{
		RunId:      runID,
		AcceptedAt: timestamppb.New(now),
	}, nil
}

// executeAsync runs scenarios in a background goroutine and persists outcome.
// ctx is a cancellable context; cancellation causes the run to stop after the
// current scenario completes and the run is marked CANCELLED.
func (r *Runner) executeAsync(runID string, req *runv1.RunAsyncRequest, ctx context.Context) {
	logger := log.FromContext(ctx)
	cfg, err := r.resolveRunConfig(ctx, req.Selector, req.Execution)
	if err != nil {
		logger.Error("failed to resolve run config", zap.Error(err))
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

	// Adapter setup runs once before the suite.
	if r.adapters != nil {
		if setupErr := r.adapters.SetupAll(ctx); setupErr != nil {
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
		if regErr := builtin.Register(reg); regErr != nil {
			logger.Error("failed to register builtin steps", zap.Error(regErr))
			r.updateRunStatus(ctx, runID, commonv1.RunStatus_RUN_STATUS_FAILED)
			return
		}
	}
	if r.hooks == nil {
		h := steps.NewHookRegistry()
		builtin.RegisterHooks(h)
		r.hooks = h
	}

	runResult := &reports.RunResult{
		RunID:     runID,
		Profile:   req.Selector.ProfileName,
		StartedAt: time.Now(),
	}
	suiteVars := mergeMaps(cfg.Variables, req.Variables)
	reporter := reports.NewConsoleReporter(nil, false, false)

	// scenarioRegex is pre-compiled in resolveRunConfig; use directly.
	scenarioRegex := cfg.CompiledScenarioRegex

	if r.hooks != nil {
		if hookErr := r.hooks.RunBeforeSuite(ctx); hookErr != nil {
			logger.Error("BeforeSuite hook failed", zap.Error(hookErr))
			r.updateRunStatus(ctx, runID, commonv1.RunStatus_RUN_STATUS_FAILED)
			return
		}
	}

	cancelled := false
	asyncScenarioOrdinal := int64(0)
	for _, feature := range features {
		if cancelled {
			break
		}
		for _, scenario := range feature.Scenarios {
			// Check if the run has been cancelled between scenarios.
			if ctx.Err() != nil {
				cancelled = true
				break
			}
			if req.Selector.TagExpression != "" && !matchesTagExpression(scenario.Tags, req.Selector.TagExpression) {
				continue
			}
			if scenarioRegex != nil && !scenarioRegex.MatchString(scenario.Name) {
				continue
			}
			if len(cfg.ScenarioIDs) > 0 && !containsString(cfg.ScenarioIDs, scenario.DeterministicID) {
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

			// Adapter reset before each scenario for clean state.
			if r.adapters != nil {
				if resetErr := r.adapters.ResetAll(ctx); resetErr != nil {
					sc.Status = reports.StatusFailed
					sc.Err = resetErr
				}
			}

			if sc.Status != reports.StatusFailed {
				if r.hooks != nil {
					if hookErr := r.hooks.RunBeforeScenario(scenCtx); hookErr != nil {
						sc.Status = reports.StatusFailed
						sc.Err = hookErr
					} else {
						r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
						if afterErr := r.hooks.RunAfterScenario(scenCtx); afterErr != nil {
							logger.Warn("AfterScenario hook error", zap.String("scenario", scenario.Name), zap.Error(afterErr))
						}
					}
				} else {
					r.executeScenario(ctx, feature, scenario, reg, scenCtx, sc, cfg, reporter)
				}
			}
			// Quarantine: demote failed non-blocking quarantined scenarios to skipped.
			if sc.Status == reports.StatusFailed && !cfg.QuarantineBlocking && scenarioIsQuarantined(scenario.Tags, cfg) {
				sc.Status = reports.StatusSkipped
			}
			asyncScenarioOrdinal++
			if r.store != nil {
				featName := feature.Name
				if upsertErr := r.store.Run.UpsertRunScenario(ctx, runstore.UpsertRunScenarioParams{
					RunID:         runID,
					ScenarioID:    sc.DeterministicID,
					Ordinal:       asyncScenarioOrdinal,
					Name:          sc.Name,
					Status:        int64(reportsStatusToScenarioProto(sc.Status)),
					DurationNanos: sc.Duration.Nanoseconds(),
					FeatureName:   &featName,
				}); upsertErr != nil {
					logger.Warn("failed to persist scenario record",
						zap.String("scenario_id", sc.DeterministicID),
						zap.Error(upsertErr))
				}
				r.persistStepDetail(ctx, runID, scenario, sc)
			}
			runResult.Scenarios = append(runResult.Scenarios, sc)
			if cfg.FailFast && sc.Status == reports.StatusFailed {
				goto asyncDone
			}
		}
	}

asyncDone:
	if r.hooks != nil {
		if afterSuiteErr := r.hooks.RunAfterSuite(ctx); afterSuiteErr != nil {
			logger.Warn("AfterSuite hook error", zap.Error(afterSuiteErr))
		}
	}
	// Adapter teardown after suite (best-effort).
	if r.adapters != nil {
		if teardownErr := r.adapters.TeardownAll(context.Background()); teardownErr != nil {
			logger.Warn("adapter teardown error", zap.Error(teardownErr))
		}
	}
	runResult.Duration = time.Since(runResult.StartedAt)
	runResult.Finalize()

	finalStatus := commonv1.RunStatus_RUN_STATUS_PASSED
	if cancelled {
		finalStatus = commonv1.RunStatus_RUN_STATUS_CANCELLED
	} else if runResult.Status == reports.StatusFailed {
		finalStatus = commonv1.RunStatus_RUN_STATUS_FAILED
	}
	r.updateRunStatus(context.Background(), runID, finalStatus)
	r.updateRunSummary(context.Background(), runID, runResult)

	if r.orchestrator != nil && !cfg.KeepStack {
		if _, teardownErr := r.orchestrator.TeardownStack(context.Background(), &stackv1.TeardownStackRequest{
			WorkspaceId: req.Selector.WorkspaceId,
		}); teardownErr != nil {
			logger.Warn("stack teardown error", zap.Error(teardownErr))
		}
	}
	// Best-effort retention pruning after run completes.
	r.pruneRetention(context.Background(), req.Selector.WorkspaceId)
}

// scenarioIsQuarantined returns true when quarantine mode is enabled and the
// scenario carries the configured quarantine tag.
func scenarioIsQuarantined(tags []string, cfg *RunConfig) bool {
	if !cfg.QuarantineEnabled {
		return false
	}
	tag := cfg.QuarantineTag
	if tag == "" {
		tag = "@quarantine"
	}
	bare := strings.TrimPrefix(tag, "@")
	for _, t := range tags {
		if t == tag || strings.TrimPrefix(t, "@") == bare {
			return true
		}
	}
	return false
}

// pruneRetention runs best-effort retention pruning for workspaceID after a
// run completes. Errors are logged but not propagated.
func (r *Runner) pruneRetention(ctx context.Context, workspaceID string) {
	if r.store == nil {
		return
	}
	if err := r.store.PruneRuns(ctx, workspaceID, r.retention); err != nil {
		log.FromContext(ctx).Warn("retention pruning failed", zap.Error(err))
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

	// Expand ${VAR} references from suite and scenario variables before
	// matching so steps like `I set the base URL to "${BASE_URL}"` resolve
	// correctly without requiring each step handler to do its own expansion.
	expandedStep := *step
	expandedStep.Text = os.Expand(step.Text, func(key string) string {
		// Check exact case first, then lowercase (Viper lowercases config keys).
		lower := strings.ToLower(key)
		for _, m := range []map[string]string{scenCtx.Variables, scenCtx.SuiteVars} {
			if v, ok := m[key]; ok {
				return v
			}
			if v, ok := m[lower]; ok {
				return v
			}
		}
		return "${" + key + "}"
	})

	stepDef, args, matchErr := reg.MatchStep(&expandedStep)
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

// configResolveTimeout is the maximum time allowed for a ConfigProvider call.
// A hung provider would otherwise block RunSync/RunAsync indefinitely.
const configResolveTimeout = 10 * time.Second

func (r *Runner) resolveRunConfig(ctx context.Context, sel *runv1.RunSelector, exec *configv1.ExecutionConfig) (*RunConfig, error) {
	if sel == nil {
		return nil, status.Error(grpccodes.InvalidArgument, "selector is required")
	}
	cfgCtx, cfgCancel := context.WithTimeout(ctx, configResolveTimeout)
	defer cfgCancel()
	cfg, err := r.configFn(cfgCtx, sel.WorkspaceId, sel.ProfileName)
	if err != nil {
		return nil, status.Errorf(grpccodes.Internal, "resolve config: %v", err)
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
	// Pre-compile the scenario name regex once here so RunSync and executeAsync
	// never recompile it on every invocation.
	if cfg.ScenarioRegex != "" {
		compiled, compileErr := regexp.Compile(cfg.ScenarioRegex)
		if compileErr != nil {
			return nil, status.Errorf(grpccodes.InvalidArgument, "invalid scenario regex: %v", compileErr)
		}
		cfg.CompiledScenarioRegex = compiled
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
			return nil, status.Errorf(grpccodes.InvalidArgument, "parse %q: %v", fp, err)
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
	if err := r.store.Run.UpdateRunStatus(ctx, runstore.UpdateRunStatusParams{
		RunID:     runID,
		Status:    int64(s),
		StartedAt: startedAt,
		EndedAt:   endedAt,
	}); err != nil {
		log.FromContext(ctx).Warn("failed to persist run status", zap.String("run_id", runID), zap.Stringer("status", s), zap.Error(err))
	}
}

func (r *Runner) updateRunSummary(ctx context.Context, runID string, res *reports.RunResult) {
	if r.store == nil {
		return
	}
	if err := r.store.Run.UpdateRunSummary(ctx, runstore.UpdateRunSummaryParams{
		RunID:                   runID,
		SummaryTotalScenarios:   int64(res.Total),
		SummaryPassedScenarios:  int64(res.Passed),
		SummaryFailedScenarios:  int64(res.Failed),
		SummarySkippedScenarios: int64(res.Skipped),
		SummaryDurationNanos:    int64(res.Duration),
	}); err != nil {
		log.FromContext(ctx).Warn("failed to persist run summary", zap.String("run_id", runID), zap.Error(err))
	}
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

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- RunEvent constructors ---

// appendEvent persists a RunEvent to the store when a store is configured.
// It silently no-ops when the store is nil.
func (r *Runner) appendEvent(ctx context.Context, runID string, evt *runv1.RunEvent) {
	if r.store == nil {
		return
	}
	p := runstore.AppendRunEventParams{
		RunID:      runID,
		Sequence:   int64(evt.Sequence),
		ObservedAt: evt.ObservedAt.AsTime().UTC().Format(time.RFC3339Nano),
		EventType:  int64(evt.EventType),
	}
	if evt.Terminal {
		p.Terminal = 1
	}
	switch pay := evt.Payload.(type) {
	case *runv1.RunEvent_RunStatus:
		v := int64(pay.RunStatus)
		p.PayloadRunStatus = &v
	case *runv1.RunEvent_ScenarioResult:
		id := pay.ScenarioResult.GetScenarioId()
		p.PayloadScenarioID = &id
	case *runv1.RunEvent_Summary:
		total := int64(pay.Summary.TotalScenarios)
		passed := int64(pay.Summary.PassedScenarios)
		failed := int64(pay.Summary.FailedScenarios)
		skipped := int64(pay.Summary.SkippedScenarios)
		durNanos := int64(0)
		if pay.Summary.Duration != nil {
			durNanos = pay.Summary.Duration.AsDuration().Nanoseconds()
		}
		p.PayloadSummaryTotalScenarios = &total
		p.PayloadSummaryPassedScenarios = &passed
		p.PayloadSummaryFailedScenarios = &failed
		p.PayloadSummarySkippedScenarios = &skipped
		p.PayloadSummaryDurationNanos = &durNanos
	}
	if err := r.store.Run.AppendRunEvent(ctx, p); err != nil {
		log.FromContext(ctx).Warn("failed to persist run event", zap.String("run_id", runID), zap.Error(err))
	}
	// Also publish to the in-process bus so live StreamRunEvents clients receive
	// events without polling the database.
	r.publishToBus(runID, evt)
}

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
	for i, step := range sc.Steps {
		stepID := fmt.Sprintf("%s:step:%d", sc.DeterministicID, i)
		protoStep := &runv1.StepResult{
			StepId:   stepID,
			Status:   stepStatusToProto(step.Status),
			Duration: durationpb.New(step.Duration),
		}
		if step.Err != nil {
			protoStep.Error = status.New(grpccodes.Unknown, step.Err.Error()).Proto()
		}
		sr.StepResults = append(sr.StepResults, protoStep)
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

// persistStepDetail writes per-step records for a completed scenario to the
// store. sc.Steps corresponds positionally to scenario.Steps (background steps
// are not included here). Errors are swallowed — a DB blip must not surface as
// a run failure.
func (r *Runner) persistStepDetail(ctx context.Context, runID string, scenario *parser.Scenario, sc *reports.ScenarioResult) {
	if r.store == nil {
		return
	}
	q := r.store.Run
	scenarioID := sc.DeterministicID

	// Scenario tags.
	for i, tag := range sc.Tags {
		_ = q.CreateRunScenarioTag(ctx, runstore.CreateRunScenarioTagParams{
			RunID:      runID,
			ScenarioID: scenarioID,
			Ordinal:    int64(i),
			Tag:        tag,
		})
	}

	// Step results.
	for i, sr := range sc.Steps {
		stepID := fmt.Sprintf("%s:step:%d", scenarioID, i)
		var errMsg *string
		if sr.Err != nil {
			s := sr.Err.Error()
			errMsg = &s
		}
		_ = q.UpsertRunStep(ctx, runstore.UpsertRunStepParams{
			RunID:         runID,
			ScenarioID:    scenarioID,
			StepID:        stepID,
			Ordinal:       int64(i),
			Keyword:       sr.Keyword,
			Text:          sr.Text,
			Status:        int64(stepStatusToProto(sr.Status)),
			DurationNanos: sr.Duration.Nanoseconds(),
			ErrorMessage:  errMsg,
		})

		if i >= len(scenario.Steps) {
			continue
		}
		// DocString.
		if scenario.Steps[i].DocString != nil {
			ds := scenario.Steps[i].DocString
			var mt *string
			if ds.MediaType != "" {
				mt = &ds.MediaType
			}
			_ = q.UpsertRunStepDocString(ctx, runstore.UpsertRunStepDocStringParams{
				RunID:       runID,
				ScenarioID:  scenarioID,
				StepID:      stepID,
				ContentType: mt,
				Content:     ds.Content,
			})
		}
		// DataTable.
		if scenario.Steps[i].DataTable != nil {
			dt := scenario.Steps[i].DataTable
			if len(dt.Rows) > 0 {
				for col, h := range dt.Rows[0] {
					_ = q.CreateRunStepDataTableHeader(ctx, runstore.CreateRunStepDataTableHeaderParams{
						RunID:      runID,
						ScenarioID: scenarioID,
						StepID:     stepID,
						Ordinal:    int64(col),
						Value:      h,
					})
				}
				for rowIdx, row := range dt.Rows[1:] {
					_ = q.CreateRunStepDataTableRow(ctx, runstore.CreateRunStepDataTableRowParams{
						RunID:      runID,
						ScenarioID: scenarioID,
						StepID:     stepID,
						RowIndex:   int64(rowIdx),
					})
					for cellIdx, cell := range row {
						_ = q.CreateRunStepDataTableCell(ctx, runstore.CreateRunStepDataTableCellParams{
							RunID:      runID,
							ScenarioID: scenarioID,
							StepID:     stepID,
							RowIndex:   int64(rowIdx),
							CellIndex:  int64(cellIdx),
							Value:      cell,
						})
					}
				}
			}
		}
	}
}
