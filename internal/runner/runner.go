// Package runner implements the gRPC RunService.Runner and PlanService.Planner
// interfaces. It coordinates Gherkin parsing, step execution, stack lifecycle,
// and result reporting for both sync and async execution modes.
package runner

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"sync"
	"time"

	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	stackv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/stack"
	"github.com/bcp-technology/lobster/internal/integrations"
	"github.com/bcp-technology/lobster/internal/reports"
	"github.com/bcp-technology/lobster/internal/steps"
	"github.com/bcp-technology/lobster/internal/store"
)

// RunConfig holds the resolved runtime configuration for a workspace run.
type RunConfig struct {
	// BaseURL is the HTTP base URL used by built-in HTTP steps.
	BaseURL string

	// DefaultHeaders are sent with every request unless overridden per-step.
	DefaultHeaders map[string]string

	// Variables are the resolved suite-scoped variable defaults.
	Variables map[string]string

	// FeaturePaths is the list of glob patterns or file paths for feature files.
	FeaturePaths []string

	// StepTimeout is the per-step execution deadline. Zero means no limit.
	StepTimeout time.Duration

	// RunTimeout is the total run execution deadline. Zero means no limit.
	RunTimeout time.Duration

	// SoftAssert collects assertion failures instead of stopping on first.
	SoftAssert bool

	// FailFast stops after the first scenario failure.
	FailFast bool

	// KeepStack prevents stack teardown after the run completes.
	KeepStack bool

	// ScenarioRegex, when non-empty, restricts execution to scenarios whose
	// name matches the regular expression.
	ScenarioRegex string

	// CompiledScenarioRegex is the compiled form of ScenarioRegex. It is set
	// by resolveRunConfig so handlers never recompile on every invocation.
	CompiledScenarioRegex *regexp.Regexp

	// ScenarioIDs, when non-empty, restricts execution to scenarios with a
	// matching DeterministicID. Populated by --from-plan.
	ScenarioIDs []string

	// QuarantineEnabled activates quarantine-tag behaviour.
	QuarantineEnabled bool

	// QuarantineTag is the tag that marks a scenario as quarantined.
	// Defaults to "@quarantine" when empty and QuarantineEnabled is true.
	QuarantineTag string

	// QuarantineBlocking, when true, quarantined failures still count as run
	// failures (useful in hardened CI profiles). When false (default), a
	// quarantined failure is demoted to StatusSkipped so the overall run
	// result is not affected.
	QuarantineBlocking bool
}

// ConfigProvider resolves runtime configuration for a workspace and profile.
type ConfigProvider func(ctx context.Context, workspaceID, profileName string) (*RunConfig, error)

// Orchestrator is the subset of the stack orchestration interface the runner
// uses for lifecycle management. *orchestration.DockerOrchestrator satisfies this.
type Orchestrator interface {
	EnsureStack(ctx context.Context, req *stackv1.EnsureStackRequest) (*stackv1.EnsureStackResponse, error)
	TeardownStack(ctx context.Context, req *stackv1.TeardownStackRequest) (*stackv1.TeardownStackResponse, error)
}

// Runner implements the runsvc.Runner interface. Each RunSync / RunAsync
// invocation creates its own isolated execution context.
type Runner struct {
	configFn     ConfigProvider
	orchestrator Orchestrator // may be nil; stack lifecycle is skipped when nil
	registry     *steps.Registry
	store        *store.Store
	hooks        *steps.HookRegistry    // may be nil; hooks are skipped when nil
	reporter     reports.Reporter       // may be nil; falls back to ConsoleReporter when nil
	adapters     *integrations.Registry // may be nil; adapter lifecycle skipped when nil
	retention    store.RetentionConfig  // zero value means no pruning

	// maxConcurrent caps the number of simultaneous RunAsync goroutines.
	// A nil channel means no limit.
	sem chan struct{}

	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc

	// busMu guards buses, the in-process event bus used by StreamRunEvents to
	// receive live events without polling the database.
	busMu sync.RWMutex
	buses map[string]chan *runv1.RunEvent
}

// New creates a Runner. orchestrator may be nil.
// maxConcurrentRuns controls how many RunAsync goroutines may run at once;
// pass 0 for no limit.
func New(cfgFn ConfigProvider, orch Orchestrator, reg *steps.Registry, st *store.Store) *Runner {
	return &Runner{
		configFn:     cfgFn,
		orchestrator: orch,
		registry:     reg,
		store:        st,
		cancels:      make(map[string]context.CancelFunc),
		buses:        make(map[string]chan *runv1.RunEvent),
	}
}

// WithMaxConcurrentRuns sets the cap on simultaneous RunAsync goroutines.
// Must be called before the first RunAsync invocation. Pass 0 for no limit.
func (r *Runner) WithMaxConcurrentRuns(n int) *Runner {
	if n > 0 {
		r.sem = make(chan struct{}, n)
	}
	return r
}

// WithHooks attaches a HookRegistry to the Runner and returns the receiver.
// Call before first use.
func (r *Runner) WithHooks(h *steps.HookRegistry) *Runner {
	r.hooks = h
	return r
}

// WithReporter injects a custom Reporter into the Runner.
// When set, it replaces the default ConsoleReporter inside RunSync / RunAsync.
// Call before first use.
func (r *Runner) WithReporter(rep reports.Reporter) *Runner {
	r.reporter = rep
	return r
}

// WithAdapterRegistry attaches an integration adapter registry to the Runner.
// SetupAll is called before each suite, ResetAll before each scenario, and
// TeardownAll after the suite. Call before first use.
func (r *Runner) WithAdapterRegistry(reg *integrations.Registry) *Runner {
	r.adapters = reg
	return r
}

// WithRetention configures run retention pruning. When workspaceID is
// non-empty and cfg has non-zero limits, PruneRuns is called (best-effort)
// after each run completes.
func (r *Runner) WithRetention(cfg store.RetentionConfig) *Runner {
	r.retention = cfg
	return r
}

// Cancel signals the background goroutine for runID to stop. It is idempotent.
func (r *Runner) Cancel(_ context.Context, runID string) error {
	r.cancelMu.Lock()
	cancel, ok := r.cancels[runID]
	if ok {
		delete(r.cancels, runID)
	}
	r.cancelMu.Unlock()
	if ok {
		cancel()
	}
	return nil
}

// SubscribeRunEvents returns the live event channel for a run that is currently
// executing in this process. The channel is closed when the run finishes.
// Returns (nil, false) when the run is not in-flight (already complete or not
// started), in which case callers should fall back to DB replay.
func (r *Runner) SubscribeRunEvents(runID string) (<-chan *runv1.RunEvent, bool) {
	r.busMu.RLock()
	ch, ok := r.buses[runID]
	r.busMu.RUnlock()
	return ch, ok
}

// openBus creates the buffered event channel for a run. Must be called before
// the first event is published so that StreamRunEvents can subscribe in time.
func (r *Runner) openBus(runID string) {
	ch := make(chan *runv1.RunEvent, 1024)
	r.busMu.Lock()
	r.buses[runID] = ch
	r.busMu.Unlock()
}

// publishToBus sends evt to the in-process bus for runID. Non-blocking: if the
// 1024-event buffer is full the event is silently dropped (this should not
// occur in practice — a backpressure condition indicates a stalled consumer).
func (r *Runner) publishToBus(runID string, evt *runv1.RunEvent) {
	r.busMu.RLock()
	ch, ok := r.buses[runID]
	r.busMu.RUnlock()
	if !ok {
		return
	}
	select {
	case ch <- evt:
	default:
		// Consumer is not keeping up; drop the live event.
		// The event is still persisted to DB so historical replay is unaffected.
	}
}

// closeBus closes and removes the channel for runID.
func (r *Runner) closeBus(runID string) {
	r.busMu.Lock()
	ch, ok := r.buses[runID]
	if ok {
		delete(r.buses, runID)
	}
	r.busMu.Unlock()
	if ok {
		close(ch)
	}
}

// Planner implements the plansvc.Planner interface.
type Planner struct {
	configFn ConfigProvider
	store    *store.Store
}

// NewPlanner creates a Planner.
func NewPlanner(cfgFn ConfigProvider, st *store.Store) *Planner {
	return &Planner{configFn: cfgFn, store: st}
}

// newUUID generates a random RFC 4122 v4 UUID string using crypto/rand.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // v4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
