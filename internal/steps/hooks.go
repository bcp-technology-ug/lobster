// Package steps — lifecycle hook registry.
//
// Hooks run at four points around test execution:
//   - BeforeSuite: once before any scenarios execute
//   - AfterSuite: once after all scenarios complete (even on failure)
//   - BeforeScenario: before each scenario
//   - AfterScenario: after each scenario
package steps

import "context"

// SuiteHookFn is a callback invoked at suite boundaries.
// ctx is the run context; returning an error fails the suite.
type SuiteHookFn func(ctx context.Context) error

// ScenarioHookFn is a callback invoked around each scenario.
// sc carries the per-scenario execution state; returning an error fails the scenario.
type ScenarioHookFn func(sc *ScenarioContext) error

// HookRegistry holds ordered lifecycle hooks registered by adapters or
// custom step packages.
type HookRegistry struct {
	beforeSuite    []SuiteHookFn
	afterSuite     []SuiteHookFn
	beforeScenario []ScenarioHookFn
	afterScenario  []ScenarioHookFn
}

// NewHookRegistry creates an empty HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// BeforeSuite registers a hook to run once before any scenarios execute.
func (h *HookRegistry) BeforeSuite(fn SuiteHookFn) {
	h.beforeSuite = append(h.beforeSuite, fn)
}

// AfterSuite registers a hook to run once after all scenarios complete.
func (h *HookRegistry) AfterSuite(fn SuiteHookFn) {
	h.afterSuite = append(h.afterSuite, fn)
}

// BeforeScenario registers a hook to run before each individual scenario.
func (h *HookRegistry) BeforeScenario(fn ScenarioHookFn) {
	h.beforeScenario = append(h.beforeScenario, fn)
}

// AfterScenario registers a hook to run after each individual scenario.
func (h *HookRegistry) AfterScenario(fn ScenarioHookFn) {
	h.afterScenario = append(h.afterScenario, fn)
}

// RunBeforeSuite invokes all registered BeforeSuite hooks in registration order.
// Returns the first error encountered; remaining hooks are not called.
func (h *HookRegistry) RunBeforeSuite(ctx context.Context) error {
	for _, fn := range h.beforeSuite {
		if err := fn(ctx); err != nil {
			return err
		}
	}
	return nil
}

// RunAfterSuite invokes all registered AfterSuite hooks in registration order.
// All hooks are called regardless of earlier errors; all errors are collected
// and the first one is returned.
func (h *HookRegistry) RunAfterSuite(ctx context.Context) error {
	var first error
	for _, fn := range h.afterSuite {
		if err := fn(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// RunBeforeScenario invokes all BeforeScenario hooks.
// Returns the first error; remaining hooks are not called.
func (h *HookRegistry) RunBeforeScenario(sc *ScenarioContext) error {
	for _, fn := range h.beforeScenario {
		if err := fn(sc); err != nil {
			return err
		}
	}
	return nil
}

// RunAfterScenario invokes all AfterScenario hooks.
// All hooks are called regardless of earlier errors; first error is returned.
func (h *HookRegistry) RunAfterScenario(sc *ScenarioContext) error {
	var first error
	for _, fn := range h.afterScenario {
		if err := fn(sc); err != nil && first == nil {
			first = err
		}
	}
	return first
}
