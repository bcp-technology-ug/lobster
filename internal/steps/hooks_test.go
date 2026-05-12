package steps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bcp-technology/lobster/internal/steps"
)

func TestHookRegistry_RunBeforeSuite_InOrder(t *testing.T) {
	t.Parallel()

	var calls []int
	h := steps.NewHookRegistry()
	h.BeforeSuite(func(_ context.Context) error { calls = append(calls, 1); return nil })
	h.BeforeSuite(func(_ context.Context) error { calls = append(calls, 2); return nil })

	if err := h.RunBeforeSuite(context.Background()); err != nil {
		t.Fatalf("RunBeforeSuite: %v", err)
	}
	if len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
		t.Errorf("calls = %v, want [1 2]", calls)
	}
}

func TestHookRegistry_RunBeforeSuite_StopsOnError(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("hook failed")
	var secondCalled bool
	h := steps.NewHookRegistry()
	h.BeforeSuite(func(_ context.Context) error { return hookErr })
	h.BeforeSuite(func(_ context.Context) error { secondCalled = true; return nil })

	err := h.RunBeforeSuite(context.Background())
	if !errors.Is(err, hookErr) {
		t.Errorf("err = %v, want hookErr", err)
	}
	if secondCalled {
		t.Error("second hook should not run after first returns error")
	}
}

func TestHookRegistry_RunAfterSuite_RunsAll(t *testing.T) {
	t.Parallel()

	err1 := errors.New("first")
	var secondCalled bool
	h := steps.NewHookRegistry()
	h.AfterSuite(func(_ context.Context) error { return err1 })
	h.AfterSuite(func(_ context.Context) error { secondCalled = true; return nil })

	err := h.RunAfterSuite(context.Background())
	if !errors.Is(err, err1) {
		t.Errorf("err = %v, want err1", err)
	}
	if !secondCalled {
		t.Error("second AfterSuite hook should run even when first errors")
	}
}

func TestHookRegistry_BeforeScenario(t *testing.T) {
	t.Parallel()

	var called bool
	h := steps.NewHookRegistry()
	h.BeforeScenario(func(sc *steps.ScenarioContext) error {
		sc.Variables["injected"] = "yes"
		called = true
		return nil
	})

	sc := steps.NewScenarioContext("", nil, nil)
	if err := h.RunBeforeScenario(sc); err != nil {
		t.Fatalf("RunBeforeScenario: %v", err)
	}
	if !called {
		t.Error("BeforeScenario hook not called")
	}
	if sc.Variables["injected"] != "yes" {
		t.Errorf("Variables[injected] = %q, want \"yes\"", sc.Variables["injected"])
	}
}

func TestHookRegistry_AfterScenario_RunsAll(t *testing.T) {
	t.Parallel()

	var calls int
	h := steps.NewHookRegistry()
	h.AfterScenario(func(_ *steps.ScenarioContext) error { calls++; return errors.New("after err") })
	h.AfterScenario(func(_ *steps.ScenarioContext) error { calls++; return nil })

	sc := steps.NewScenarioContext("", nil, nil)
	err := h.RunAfterScenario(sc)
	if err == nil {
		t.Error("expected error from first hook")
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (all hooks must run)", calls)
	}
}

func TestHookRegistry_NoHooks_NoError(t *testing.T) {
	t.Parallel()

	h := steps.NewHookRegistry()
	ctx := context.Background()
	sc := steps.NewScenarioContext("", nil, nil)

	if err := h.RunBeforeSuite(ctx); err != nil {
		t.Errorf("RunBeforeSuite: %v", err)
	}
	if err := h.RunAfterSuite(ctx); err != nil {
		t.Errorf("RunAfterSuite: %v", err)
	}
	if err := h.RunBeforeScenario(sc); err != nil {
		t.Errorf("RunBeforeScenario: %v", err)
	}
	if err := h.RunAfterScenario(sc); err != nil {
		t.Errorf("RunAfterScenario: %v", err)
	}
}
