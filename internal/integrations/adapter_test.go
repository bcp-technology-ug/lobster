// Package integrations_test provides black-box tests for the adapter Registry
// and Validator.
package integrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bcp-technology/lobster/internal/integrations"
)

// --- fake adapter used in tests ---

type fakeAdapter struct {
	id       string
	kind     string
	setupErr error
	resetErr error
	tearErr  error
	setupN   int
	resetN   int
	tearN    int
}

func (f *fakeAdapter) ID() string   { return f.id }
func (f *fakeAdapter) Kind() string { return f.kind }
func (f *fakeAdapter) Setup(_ context.Context) error {
	f.setupN++
	return f.setupErr
}
func (f *fakeAdapter) Reset(_ context.Context) error {
	f.resetN++
	return f.resetErr
}
func (f *fakeAdapter) Teardown(_ context.Context) error {
	f.tearN++
	return f.tearErr
}

// --- Registry tests ---

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	fa := &fakeAdapter{id: "a", kind: "fake"}
	if err := reg.Register(fa); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := reg.Get("a")
	if !ok {
		t.Fatal("Get: not found")
	}
	if got.ID() != "a" {
		t.Errorf("ID = %q, want %q", got.ID(), "a")
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	fa := &fakeAdapter{id: "dup", kind: "fake"}
	_ = reg.Register(fa)
	err := reg.Register(fa)
	if err == nil {
		t.Fatal("expected error for duplicate register, got nil")
	}
}

func TestRegistry_GetMissing(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Get returned ok=true for missing adapter")
	}
}

func TestRegistry_All(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	_ = reg.Register(&fakeAdapter{id: "x", kind: "fake"})
	_ = reg.Register(&fakeAdapter{id: "y", kind: "fake"})

	all := reg.All()
	if len(all) != 2 {
		t.Errorf("len(All) = %d, want 2", len(all))
	}
}

func TestRegistry_SetupAll_Success(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	fa := &fakeAdapter{id: "s", kind: "fake"}
	_ = reg.Register(fa)

	if err := reg.SetupAll(context.Background()); err != nil {
		t.Fatalf("SetupAll: %v", err)
	}
	if fa.setupN != 1 {
		t.Errorf("Setup called %d times, want 1", fa.setupN)
	}
}

func TestRegistry_SetupAll_StopsOnFirstError(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	fa1 := &fakeAdapter{id: "fail", kind: "fake", setupErr: errors.New("boom")}
	_ = reg.Register(fa1)

	err := reg.SetupAll(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegistry_ResetAll_RunsAll(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	fa1 := &fakeAdapter{id: "r1", kind: "fake"}
	fa2 := &fakeAdapter{id: "r2", kind: "fake", resetErr: errors.New("reset fail")}
	_ = reg.Register(fa1)
	_ = reg.Register(fa2)

	// ResetAll should continue even if one fails and collect errors.
	err := reg.ResetAll(context.Background())
	if err == nil {
		t.Error("expected error from failed reset, got nil")
	}
	// Both adapters should have been called (run-all semantics).
	if fa1.resetN == 0 {
		t.Error("first adapter Reset was not called")
	}
	if fa2.resetN == 0 {
		t.Error("second adapter Reset was not called")
	}
}

func TestRegistry_TeardownAll_RunsAll(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	fa1 := &fakeAdapter{id: "t1", kind: "fake"}
	fa2 := &fakeAdapter{id: "t2", kind: "fake", tearErr: errors.New("tear fail")}
	_ = reg.Register(fa1)
	_ = reg.Register(fa2)

	err := reg.TeardownAll(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
	if fa1.tearN == 0 {
		t.Error("first adapter Teardown was not called")
	}
	if fa2.tearN == 0 {
		t.Error("second adapter Teardown was not called")
	}
}

// --- Validator tests ---

func TestValidator_Validate_Found(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	_ = reg.Register(&fakeAdapter{id: "kc", kind: "keycloak"})

	v := integrations.NewValidator(reg)
	ok, err := v.Validate(context.Background(), "kc")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !ok {
		t.Error("Validate = false, want true")
	}
}

func TestValidator_Validate_NotFound(t *testing.T) {
	t.Parallel()

	reg := integrations.NewRegistry()
	v := integrations.NewValidator(reg)
	ok, err := v.Validate(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if ok {
		t.Error("Validate = true for missing adapter, want false")
	}
}
