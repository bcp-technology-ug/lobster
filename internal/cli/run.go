package cli

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	configv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/config"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology-ug/lobster/internal/integrations"
	"github.com/bcp-technology-ug/lobster/internal/integrations/keycloak"
	"github.com/bcp-technology-ug/lobster/internal/orchestration"
	"github.com/bcp-technology-ug/lobster/internal/parser"
	"github.com/bcp-technology-ug/lobster/internal/reports"
	"github.com/bcp-technology-ug/lobster/internal/runner"
	"github.com/bcp-technology-ug/lobster/internal/steps"
	"github.com/bcp-technology-ug/lobster/internal/steps/builtin"
	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/telemetry"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

// newRunCommand creates the `lobster run` command with full runner wiring.
func newRunCommand(v *viper.Viper) *cobra.Command {
	var verbosity int

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute scenarios against configured stack",
		Long:  "Execute all matched scenarios against the configured Docker Compose stack.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCommand(cmd, v, verbosity)
		},
	}

	// Feature selection.
	cmd.Flags().String("features", "", "feature file glob pattern (e.g. features/**/*.feature)")
	cmd.Flags().StringSlice("tags", nil, "tag filter expression (e.g. @smoke)")
	cmd.Flags().String("scenario-regex", "", "run only scenarios whose name matches this regex")
	cmd.Flags().String("from-plan", "", "execute from a saved plan artifact (path to plan JSON file)")

	// Infrastructure.
	cmd.Flags().StringSlice("compose", nil, "docker-compose file(s)")
	cmd.Flags().StringSlice("compose-profile", nil, "compose profile(s) to activate")
	cmd.Flags().Bool("no-compose", false, "skip stack orchestration even if compose files are configured")

	// Execution options.
	cmd.Flags().Bool("fail-fast", false, "stop after first scenario failure")
	cmd.Flags().Bool("soft-assert", false, "collect all assertion failures instead of stopping on first")
	cmd.Flags().Bool("keep-stack", false, "skip infrastructure teardown after run")
	cmd.Flags().Duration("timeout", 0, "maximum overall run duration (e.g. 10m)")
	cmd.Flags().Duration("step-timeout", 0, "maximum duration per step (e.g. 30s)")
	cmd.Flags().StringArray("env", nil, "variable override KEY=VALUE (repeatable)")

	// Output / verbosity.
	cmd.Flags().Bool("ci", false, "force non-interactive CI output (plain text)")
	cmd.Flags().CountVarP(&verbosity, "verbose", "v", "increase output verbosity: -v info, -vv debug, -vvv trace")
	cmd.Flags().Bool("report-verbose", false, "include step-level detail in console and CI output (mirrors reports.verbose in config)")
	cmd.Flags().String("report-json", "", "write JSON report to path")
	cmd.Flags().String("report-junit", "", "write JUnit XML report to path")

	// Run mode.
	cmd.Flags().String("run-mode", "sync", "execution mode: sync|async (async requires daemon)")

	// Executor mode and daemon connection.
	cmd.Flags().String("executor-mode", "local", "executor target: local|daemon")
	cmd.PersistentFlags().String("executor-addr", "", "daemon gRPC address (e.g. dns:///lobsterd:9443)")
	cmd.PersistentFlags().String("auth-token", "", "bearer token for daemon authentication")
	cmd.PersistentFlags().String("tls-ca-file", "", "CA certificate file for daemon TLS")
	cmd.PersistentFlags().String("tls-cert-file", "", "client certificate file for daemon mTLS")
	cmd.PersistentFlags().String("tls-key-file", "", "client key file for daemon mTLS")

	// Observability.
	cmd.Flags().String("otel-endpoint", "", "OpenTelemetry collector endpoint (e.g. http://localhost:4318)")
	cmd.Flags().String("otel-service-name", "lobster", "service name for OTel traces")

	// Persistence.
	addPersistenceFlags(cmd.Flags())

	// Sub-commands for async run lifecycle.
	cmd.AddCommand(newRunWatchCommand(v))
	cmd.AddCommand(newRunStatusCommand(v))
	cmd.AddCommand(newRunCancelCommand(v))

	return cmd
}

// runCommand is the core implementation of `lobster run`.
func runCommand(cmd *cobra.Command, v *viper.Viper, verbosity int) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// ── feature paths ──────────────────────────────────────────────────────
	featureFlag, _ := cmd.Flags().GetString("features")
	featuresCfg := v.GetStringSlice("features.paths")
	if featureFlag != "" {
		featuresCfg = []string{featureFlag}
	}
	if len(featuresCfg) == 0 {
		featuresCfg = []string{"features/**/*.feature"}
	}

	// ── from-plan: override feature paths and add scenario ID filter ────────
	fromPlan, _ := cmd.Flags().GetString("from-plan")
	var planScenarioIDs []string
	if fromPlan != "" {
		p, err := loadPlanFile(fromPlan)
		if err != nil {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Invalid plan file", err.Error(),
				"Provide a path to a file saved with 'lobster plan --out <file>'", ""))
			return &ExitError{Code: ExitConfigError}
		}
		if p.FeaturePath != "" {
			featuresCfg = []string{p.FeaturePath}
		}
		planScenarioIDs = p.ScenarioIDs
	}

	// ── tags ────────────────────────────────────────────────────────────────
	tags, _ := cmd.Flags().GetStringSlice("tags")
	tagExpr := strings.Join(tags, ",")
	if tagExpr == "" {
		tagExpr = v.GetString("runner.tags")
	}

	// ── scenario regex ──────────────────────────────────────────────────────
	scenarioRegex, _ := cmd.Flags().GetString("scenario-regex")

	// ── env overrides ───────────────────────────────────────────────────────
	envList, _ := cmd.Flags().GetStringArray("env")
	envMap := make(map[string]string, len(envList))
	for _, e := range envList {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Invalid --env value",
				fmt.Sprintf("%q is not in KEY=VALUE format", e),
				"Example: --env BASE_URL=http://localhost:8080", ""))
			return &ExitError{Code: ExitConfigError}
		}
		envMap[parts[0]] = parts[1]
	}

	// ── execution flags ─────────────────────────────────────────────────────
	failFast, _ := cmd.Flags().GetBool("fail-fast")
	softAssert, _ := cmd.Flags().GetBool("soft-assert")
	keepStack, _ := cmd.Flags().GetBool("keep-stack")
	ciMode, _ := cmd.Flags().GetBool("ci")
	runMode, _ := cmd.Flags().GetString("run-mode")
	executorMode, _ := cmd.Flags().GetString("executor-mode")
	executorAddr, _ := cmd.Flags().GetString("executor-addr")
	authToken, _ := cmd.Flags().GetString("auth-token")
	tlsCA, _ := cmd.Flags().GetString("tls-ca-file")
	tlsCert, _ := cmd.Flags().GetString("tls-cert-file")
	tlsKey, _ := cmd.Flags().GetString("tls-key-file")

	if !cmd.Flags().Changed("fail-fast") && v.IsSet("runner.fail_fast") {
		failFast = v.GetBool("runner.fail_fast")
	}
	if !cmd.Flags().Changed("soft-assert") && v.IsSet("runner.soft_assert") {
		softAssert = v.GetBool("runner.soft_assert")
	}
	if !cmd.Flags().Changed("keep-stack") && v.IsSet("runner.keep_stack") {
		keepStack = v.GetBool("runner.keep_stack")
	}

	// ── timeouts ────────────────────────────────────────────────────────────
	runTimeout, _ := cmd.Flags().GetDuration("timeout")
	stepTimeout, _ := cmd.Flags().GetDuration("step-timeout")
	if runTimeout == 0 {
		runTimeout = v.GetDuration("runner.run_timeout")
	}
	if stepTimeout == 0 {
		stepTimeout = v.GetDuration("runner.step_timeout")
	}
	if runTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, runTimeout)
		defer cancel()
	}

	// ── report paths & verbosity ────────────────────────────────────────────
	reportJSON, _ := cmd.Flags().GetString("report-json")
	reportJUnit, _ := cmd.Flags().GetString("report-junit")
	reportVerbose, _ := cmd.Flags().GetBool("report-verbose")
	if !cmd.Flags().Changed("report-verbose") && v.IsSet("reports.verbose") {
		reportVerbose = v.GetBool("reports.verbose")
	}

	// ── run request ─────────────────────────────────────────────────────────
	workspaceID := v.GetString("workspace.selected")
	if workspaceID == "" {
		workspaceID = "default"
	}
	req := &runv1.RunSyncRequest{
		Selector: &runv1.RunSelector{
			WorkspaceId:   workspaceID,
			TagExpression: tagExpr,
		},
		Execution: &configv1.ExecutionConfig{
			FailFast:   failFast,
			SoftAssert: softAssert,
			KeepStack:  keepStack,
		},
	}
	if len(featuresCfg) == 1 {
		req.Selector.FeaturePath = featuresCfg[0]
	}
	if stepTimeout > 0 {
		req.Execution.StepTimeout = durationpb.New(stepTimeout)
	}

	// ── validate run-mode / executor-mode combination ────────────────────────
	if runMode == "async" && executorMode != "daemon" {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Invalid flag combination",
			"--run-mode async requires --executor-mode daemon",
			"Start the lobsterd daemon and set --executor-addr, then retry.", ""))
		return &ExitError{Code: ExitConfigError}
	}

	// ── daemon execution path ─────────────────────────────────────────────
	if executorMode == "daemon" {
		if executorAddr == "" {
			executorAddr = v.GetString("daemon.addr")
		}
		if executorAddr == "" {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Missing daemon address",
				"--executor-addr is required when --executor-mode daemon is set",
				"Example: --executor-addr dns:///lobsterd:9443", ""))
			return &ExitError{Code: ExitConfigError}
		}

		conn, err := dialDaemon(ctx, executorAddr, authToken, tlsCA, tlsCert, tlsKey)
		if err != nil {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed", err.Error(),
				"Check --executor-addr and TLS/auth flags.", ""))
			return &ExitError{Code: ExitOrchestration}
		}
		defer conn.Close()

		client := runv1.NewRunServiceClient(conn)

		if runMode == "async" {
			return runWithDaemonAsync(ctx, client, req, cmd)
		}
		return runWithDaemonSync(ctx, client, req, reportJSON, reportJUnit, cmd)
	}

	// ── interactive scenario picker ──────────────────────────────────────────
	// Show the picker when in local mode, no explicit scenario filter is set,
	// and we are attached to a real TTY.
	interactive := ui.IsInteractive() && !ciMode
	if interactive && tagExpr == "" && scenarioRegex == "" && len(planScenarioIDs) == 0 && executorMode != "daemon" {
		var features []*parser.Feature
		for _, glob := range featuresCfg {
			parsed, parseErr := parser.ParseGlob(glob)
			if parseErr == nil {
				features = append(features, parsed...)
			}
		}
		if len(features) > 0 {
			picker := ui.NewScenarioPickerModel(features)
			p := tea.NewProgram(picker)
			finalModel, runErr2 := p.Run()
			if runErr2 == nil {
				if pm, ok := finalModel.(ui.ScenarioPickerModel); ok {
					if pm.Done() {
						if ids := pm.SelectedIDs(); len(ids) > 0 {
							planScenarioIDs = ids
						}
					}
				}
			}
		}
	}

	// ── local execution path ─────────────────────────────────────────────

	// OTel: initialise tracing when an endpoint is configured.
	otelEndpoint, _ := cmd.Flags().GetString("otel-endpoint")
	if otelEndpoint == "" {
		otelEndpoint = v.GetString("telemetry.otel.endpoint")
	}
	otelServiceName, _ := cmd.Flags().GetString("otel-service-name")
	if otelServiceName == "" || otelServiceName == "lobster" {
		if s := v.GetString("telemetry.otel.service_name"); s != "" {
			otelServiceName = s
		}
	}
	tprov, otelErr := telemetry.Setup(ctx, telemetry.Config{
		Endpoint:    otelEndpoint,
		ServiceName: otelServiceName,
	})
	if otelErr != nil {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("OpenTelemetry setup failed",
			otelErr.Error(), "Check --otel-endpoint and ensure the collector is reachable.", ""))
		return &ExitError{Code: ExitRuntimeError}
	}
	defer func() { _ = tprov.Shutdown(context.Background()) }()

	// Config provider merges viper config, then applies --env overrides.
	baseURL := v.GetString("http.base_url")
	configVariables := v.GetStringMapString("variables")
	configFn := func(_ context.Context, _, _ string) (*runner.RunConfig, error) {
		vars := make(map[string]string, len(configVariables)+len(envMap))
		for k, val := range configVariables {
			vars[k] = val
		}
		for k, val := range envMap {
			vars[k] = val
		}
		return &runner.RunConfig{
			BaseURL:            baseURL,
			FeaturePaths:       featuresCfg,
			Variables:          vars,
			FailFast:           failFast,
			SoftAssert:         softAssert,
			KeepStack:          keepStack,
			StepTimeout:        stepTimeout,
			ScenarioRegex:      scenarioRegex,
			ScenarioIDs:        planScenarioIDs,
			QuarantineEnabled:  v.GetBool("quarantine.enabled"),
			QuarantineTag:      v.GetString("quarantine.tag"),
			QuarantineBlocking: v.GetBool("quarantine.blocking_in_main_ci"),
		}, nil
	}

	// Orchestrator.
	var orch runner.Orchestrator
	noCompose, _ := cmd.Flags().GetBool("no-compose")
	composePaths, _ := cmd.Flags().GetStringSlice("compose")
	composeProfiles, _ := cmd.Flags().GetStringSlice("compose-profile")
	if len(composePaths) == 0 {
		composePaths = v.GetStringSlice("compose.files")
	}
	if len(composePaths) > 0 && !noCompose {
		orchSetup := &orchestration.Setup{
			ComposeFiles: composePaths,
			Profiles:     composeProfiles,
		}
		orchInstance, err := orchestration.New("", func(_ context.Context, _, _ string) (*orchestration.Setup, error) {
			return orchSetup, nil
		})
		if err != nil {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Orchestration error",
				fmt.Sprintf("create docker orchestrator: %v", err), "", ""))
			return &ExitError{Code: ExitOrchestration}
		}
		orch = orchInstance
	}

	// Store.
	storeConfig, err := buildStoreConfigFromInputs(cmd, v)
	if err != nil {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Configuration error", err.Error(), "", ""))
		return &ExitError{Code: ExitConfigError}
	}
	var st *store.Store
	if storeConfig.SQLitePath != "" {
		st, err = store.Open(ctx, storeConfig)
		if err != nil {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Store error",
				fmt.Sprintf("open store: %v", err), "", ""))
			return &ExitError{Code: ExitRuntimeError}
		}
		defer st.Close()
	}

	// Step registry.
	reg := steps.NewRegistry()
	if err := builtin.Register(reg); err != nil {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Internal error",
			fmt.Sprintf("register builtin steps: %v", err), "", ""))
		return &ExitError{Code: ExitRuntimeError}
	}

	r := runner.New(configFn, orch, reg, st)

	hooks := steps.NewHookRegistry()
	builtin.RegisterHooks(hooks)
	r = r.WithHooks(hooks)

	// Apply retention config from persistence settings.
	retentionCfg := store.RetentionConfig{
		MaxRuns: int64(v.GetInt("persistence.retention.max_runs")),
		MaxAge:  v.GetDuration("persistence.retention.max_age"),
	}
	if retentionCfg.MaxRuns > 0 || retentionCfg.MaxAge > 0 {
		r = r.WithRetention(retentionCfg)
	}

	// Integrations (optional; keycloak is the only supported adapter in v0.1).
	intReg := integrations.NewRegistry()
	if v.GetBool("integrations.keycloak.enabled") {
		adminPassword := os.Getenv(v.GetString("integrations.keycloak.admin_password_env"))
		kcAdapter := keycloak.New("keycloak-primary", keycloak.Config{
			BaseURL:       v.GetString("integrations.keycloak.base_url"),
			AdminUser:     v.GetString("integrations.keycloak.admin_user"),
			AdminPassword: adminPassword,
			Realm:         v.GetString("integrations.keycloak.realm"),
		})
		if regErr := intReg.Register(kcAdapter); regErr != nil {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Internal error",
				fmt.Sprintf("register keycloak adapter: %v", regErr), "", ""))
			return &ExitError{Code: ExitRuntimeError}
		}
		if regErr := kcAdapter.RegisterSteps(reg); regErr != nil {
			_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Internal error",
				fmt.Sprintf("register keycloak steps: %v", regErr), "", ""))
			return &ExitError{Code: ExitRuntimeError}
		}
	}
	r = r.WithAdapterRegistry(intReg)

	// Side-car reporters (JSON, JUnit) + capture reporter for undefined steps.
	var extraReporters []reports.Reporter
	if reportJSON != "" {
		extraReporters = append(extraReporters, reports.NewJSONReporter(reportJSON))
	}
	if reportJUnit != "" {
		extraReporters = append(extraReporters, reports.NewJUnitReporter(reportJUnit))
	}
	capture := &ui.CaptureReporter{}
	extraReporters = append(extraReporters, capture)

	// Choose TUI or console.
	var runErr error
	if interactive {
		runErr = runWithTUI(ctx, r, req, extraReporters, verbosity, cmd)
	} else {
		runErr = runWithConsole(ctx, r, req, extraReporters, verbosity, reportVerbose, cmd)
	}

	// Print undefined steps summary.
	if capture.Result != nil {
		printUndefinedStepsSummary(cmd, capture.Result)
	}

	// In CI/console mode, RunSync does not propagate scenario failures as
	// errors. Check the captured result to ensure we exit non-zero.
	if runErr == nil && capture.Result != nil && capture.Result.Failed > 0 {
		return &ExitError{Code: ExitScenarioFailure}
	}

	return runErr
}

// runWithTUI runs the test suite with a Bubbletea live-updating TUI.
func runWithTUI(
	ctx context.Context,
	r *runner.Runner,
	req *runv1.RunSyncRequest,
	extra []reports.Reporter,
	_ int,
	cmd *cobra.Command,
) error {
	model := ui.NewRunModel()
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))

	tuiReporter := ui.NewTUIReporter(p)

	allReporters := []reports.Reporter{tuiReporter}
	allReporters = append(allReporters, extra...)
	r = r.WithReporter(reports.NewMultiReporter(allReporters...))

	runErrCh := make(chan error, 1)
	go func() {
		stream := &noopStream{ctx: ctx}
		err := r.RunSync(ctx, req, stream)
		runErrCh <- err
		if err != nil {
			p.Send(ui.RunnerErrMsg{Err: err})
		}
	}()

	finalModel, tuiErr := p.Run()
	if tuiErr != nil {
		return tuiErr
	}

	runErr := <-runErrCh
	if runErr != nil {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), ui.RenderError("Run failed",
			runErr.Error(), "", ""))
		return &ExitError{Code: exitCodeForRunError(runErr)}
	}

	if fm, ok := finalModel.(ui.RunModel); ok {
		if s := fm.Summary(); s != nil && s.Failed > 0 {
			return &ExitError{Code: ExitScenarioFailure}
		}
	}
	return nil
}

// runWithConsole runs the test suite with plain console output (CI / non-TTY).
func runWithConsole(
	ctx context.Context,
	r *runner.Runner,
	req *runv1.RunSyncRequest,
	extra []reports.Reporter,
	verbosity int,
	reportVerbose bool,
	cmd *cobra.Command,
) error {
	consoleReporter := reports.NewConsoleReporter(cmd.OutOrStdout(), verbosity >= 1 || reportVerbose, true)

	allReporters := []reports.Reporter{consoleReporter}
	allReporters = append(allReporters, extra...)
	r = r.WithReporter(reports.NewMultiReporter(allReporters...))

	stream := &noopStream{ctx: ctx}
	if err := r.RunSync(ctx, req, stream); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, ui.RenderError("Run failed", err.Error(), "", ""))
		return &ExitError{Code: exitCodeForRunError(err)}
	}
	return nil
}

// runWithDaemonSync streams RunSync events from the daemon and renders them.
func runWithDaemonSync(
	ctx context.Context,
	client runv1.RunServiceClient,
	req *runv1.RunSyncRequest,
	reportJSONPath, reportJUnitPath string,
	cmd *cobra.Command,
) error {
	stream, err := client.RunSync(ctx, req)
	if err != nil {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon run failed",
			err.Error(), "", ""))
		return &ExitError{Code: exitCodeForRunError(err)}
	}

	runResult := &reports.RunResult{StartedAt: time.Now()}
	var failed bool
	for {
		ev, recvErr := stream.Recv()
		if recvErr != nil {
			break
		}
		if runResult.RunID == "" {
			runResult.RunID = ev.GetRunId()
		}
		switch p := ev.GetPayload().(type) {
		case *runv1.RunEvent_RunStatus:
			if p.RunStatus == commonv1.RunStatus_RUN_STATUS_RUNNING {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(),
					ui.StyleMuted.Render("▶  run started  ["+ev.GetRunId()+"]"))
			}
		case *runv1.RunEvent_ScenarioResult:
			sc := p.ScenarioResult
			scResult := &reports.ScenarioResult{
				DeterministicID: sc.GetScenarioId(),
				Status:          protoScenarioStatusToReports(sc.GetStatus()),
			}
			if dur := sc.GetDuration(); dur != nil {
				scResult.Duration = dur.AsDuration()
			}
			runResult.Scenarios = append(runResult.Scenarios, scResult)
			if scResult.Status == reports.StatusFailed {
				failed = true
				_, _ = fmt.Fprintln(cmd.OutOrStdout(),
					ui.StyleError.Render(ui.IconCross+" "+sc.GetScenarioId()))
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(),
					ui.StyleSuccess.Render(ui.IconCheck+" "+sc.GetScenarioId()))
			}
		case *runv1.RunEvent_Summary:
			s := p.Summary
			line := fmt.Sprintf("total=%d  passed=%d  failed=%d  skipped=%d",
				s.GetTotalScenarios(), s.GetPassedScenarios(),
				s.GetFailedScenarios(), s.GetSkippedScenarios())
			if s.GetFailedScenarios() > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleError.Render(ui.IconCross+"  "+line))
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleSuccess.Render(ui.IconCheck+"  "+line))
			}
		}
		if ev.GetTerminal() {
			break
		}
	}

	runResult.Duration = time.Since(runResult.StartedAt)
	runResult.Finalize()

	if reportJSONPath != "" {
		rep := reports.NewJSONReporter(reportJSONPath)
		rep.RunFinished(runResult)
	}
	if reportJUnitPath != "" {
		rep := reports.NewJUnitReporter(reportJUnitPath)
		rep.RunFinished(runResult)
	}

	if failed {
		return &ExitError{Code: ExitScenarioFailure}
	}
	return nil
}

// protoScenarioStatusToReports converts a proto ScenarioStatus to reports.Status.
func protoScenarioStatusToReports(s commonv1.ScenarioStatus) reports.Status {
	switch s {
	case commonv1.ScenarioStatus_SCENARIO_STATUS_PASSED:
		return reports.StatusPassed
	case commonv1.ScenarioStatus_SCENARIO_STATUS_FAILED:
		return reports.StatusFailed
	case commonv1.ScenarioStatus_SCENARIO_STATUS_SKIPPED:
		return reports.StatusSkipped
	default:
		return reports.StatusUndefined
	}
}

// runWithDaemonAsync submits an async run and prints the run ID.
func runWithDaemonAsync(
	ctx context.Context,
	client runv1.RunServiceClient,
	req *runv1.RunSyncRequest,
	cmd *cobra.Command,
) error {
	// Generate a unique idempotency key for this submission.
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	idempKey := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	asyncReq := &runv1.RunAsyncRequest{
		Selector:       req.Selector,
		Execution:      req.Execution,
		IdempotencyKey: idempKey,
	}
	resp, err := client.RunAsync(ctx, asyncReq)
	if err != nil {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon async run failed",
			err.Error(), "", ""))
		return &ExitError{Code: exitCodeForRunError(err)}
	}
	// Print only the run ID to stdout so it can be captured by scripts/tests.
	// Human-friendly guidance goes to stderr to avoid polluting the output.
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), resp.GetRunId())
	_, _ = fmt.Fprint(cmd.ErrOrStderr(),
		ui.RenderSuccess("Run submitted", "Run ID: "+resp.GetRunId()))
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
		ui.StyleMuted.Render("Use 'lobster run watch --run-id "+resp.GetRunId()+"' to follow progress."))
	return nil
}

// exitCodeForRunError maps a gRPC status error to a differentiated exit code.
func exitCodeForRunError(err error) int {
	if err == nil {
		return ExitScenarioFailure
	}
	st, ok := grpcstatus.FromError(err)
	if !ok {
		return ExitRuntimeError
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return ExitConfigError
	case codes.Internal:
		msg := st.Message()
		if strings.Contains(msg, "ensure stack") || strings.Contains(msg, "docker") {
			return ExitOrchestration
		}
		return ExitRuntimeError
	default:
		return ExitRuntimeError
	}
}

// printUndefinedStepsSummary prints a grouped list of unique undefined steps.
func printUndefinedStepsSummary(cmd *cobra.Command, r *reports.RunResult) {
	type stepKey struct{ keyword, text string }
	seen := make(map[stepKey]struct{})
	var undefs []stepKey

	for _, sc := range r.Scenarios {
		for _, sr := range sc.Steps {
			if sr.Status == reports.StatusUndefined {
				k := stepKey{keyword: sr.Keyword, text: sr.Text}
				if _, ok := seen[k]; !ok {
					seen[k] = struct{}{}
					undefs = append(undefs, k)
				}
			}
		}
	}
	if len(undefs) == 0 {
		return
	}

	rows := make([][2]string, 0, len(undefs))
	for _, u := range undefs {
		rows = append(rows, [2]string{
			ui.StyleWarning.Render(ui.IconWarning + " undefined"),
			ui.StyleCode.Render(u.keyword + u.text),
		})
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), "\n")
	_, _ = fmt.Fprint(cmd.OutOrStdout(),
		ui.RenderKeyValueTable(fmt.Sprintf("Undefined steps (%d)", len(undefs)), rows))
}

// ── plan file ──────────────────────────────────────────────────────────────

// planFile is the JSON schema written by `lobster plan --out`.
type planFile struct {
	PlanID      string   `json:"plan_id,omitempty"`
	FeaturePath string   `json:"feature_path,omitempty"`
	ScenarioIDs []string `json:"scenario_ids,omitempty"`
}

func loadPlanFile(path string) (*planFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	var p planFile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse plan JSON: %w", err)
	}
	return &p, nil
}

// keep time import used
var _ = time.Second
