// Package runner implements the gRPC RunService.Runner and PlanService.Planner
// interfaces. It coordinates Gherkin parsing, step execution, stack lifecycle,
// and result reporting for both sync and async execution modes.
package runner

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	stackv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/stack"
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

	// ScenarioIDs, when non-empty, restricts execution to scenarios with a
	// matching DeterministicID. Populated by --from-plan.
	ScenarioIDs []string
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
	hooks        *steps.HookRegistry // may be nil; hooks are skipped when nil
	reporter     reports.Reporter    // may be nil; falls back to ConsoleReporter when nil
}

// New creates a Runner. orchestrator may be nil.
func New(cfgFn ConfigProvider, orch Orchestrator, reg *steps.Registry, st *store.Store) *Runner {
	return &Runner{
		configFn:     cfgFn,
		orchestrator: orch,
		registry:     reg,
		store:        st,
	}
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
