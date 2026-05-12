// Package integrations provides the integration adapter lifecycle model and
// the Registry that wires adapters into the execution pipeline.
//
// Adapters are registered statically in v0.1. Each adapter implements
// setup/reset/teardown hooks that run around suite and scenario boundaries.
package integrations

import (
	"context"
	"fmt"
	"sync"
)

// Adapter is the lifecycle interface every integration adapter must implement.
type Adapter interface {
	// ID returns the adapter's unique identifier (e.g. "keycloak-primary").
	ID() string

	// Kind returns the adapter type (e.g. "keycloak", "custom").
	Kind() string

	// Setup prepares required entities before suite execution begins.
	Setup(ctx context.Context) error

	// Reset restores known state between scenarios (e.g. reseeds test data).
	Reset(ctx context.Context) error

	// Teardown removes transient artifacts after suite execution ends.
	Teardown(ctx context.Context) error
}

// Registry holds the registered adapters for a run.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

// Register adds an adapter to the registry. Returns an error if an adapter
// with the same ID is already registered.
func (r *Registry) Register(a Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.adapters[a.ID()]; ok {
		return fmt.Errorf("adapter %q already registered", a.ID())
	}
	r.adapters[a.ID()] = a
	return nil
}

// Get returns the adapter for the given ID, or (nil, false) if not found.
func (r *Registry) Get(id string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[id]
	return a, ok
}

// All returns all registered adapters in arbitrary order.
func (r *Registry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		out = append(out, a)
	}
	return out
}

// SetupAll calls Setup on every registered adapter. Stops and returns the first
// error.
func (r *Registry) SetupAll(ctx context.Context) error {
	for _, a := range r.All() {
		if err := a.Setup(ctx); err != nil {
			return fmt.Errorf("adapter %q setup: %w", a.ID(), err)
		}
	}
	return nil
}

// ResetAll calls Reset on every registered adapter, collecting errors.
func (r *Registry) ResetAll(ctx context.Context) error {
	var errs []error
	for _, a := range r.All() {
		if err := a.Reset(ctx); err != nil {
			errs = append(errs, fmt.Errorf("adapter %q reset: %w", a.ID(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("reset errors: %v", errs)
	}
	return nil
}

// TeardownAll calls Teardown on every registered adapter, collecting errors.
func (r *Registry) TeardownAll(ctx context.Context) error {
	var errs []error
	for _, a := range r.All() {
		if err := a.Teardown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("adapter %q teardown: %w", a.ID(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("teardown errors: %v", errs)
	}
	return nil
}

// Validator implements the integrationsvc.AdapterValidator interface.
// It reports whether an adapter ID is registered and reachable.
type Validator struct {
	registry *Registry
}

// NewValidator creates a Validator backed by the given registry.
func NewValidator(reg *Registry) *Validator {
	return &Validator{registry: reg}
}

// Validate returns true if an adapter with the given ID is registered.
// Context is accepted for future remote-validation support.
func (v *Validator) Validate(_ context.Context, adapterID string) (bool, error) {
	_, ok := v.registry.Get(adapterID)
	return ok, nil
}
