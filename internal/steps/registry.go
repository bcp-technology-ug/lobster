// Package steps provides the step definition registry and scenario execution
// context for Lobster BDD test execution.
package steps

import (
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/bcp-technology/lobster/internal/parser"
)

// ErrUndefined is returned when no registered step matches the step text.
var ErrUndefined = errors.New("undefined step")

// StepHandler is a function that executes a step.
// ctx carries per-scenario state. args contains the regexp capture groups
// extracted from the step text.
type StepHandler func(ctx *ScenarioContext, args ...string) error

// StepDef is a compiled step definition: pattern + handler.
type StepDef struct {
	Pattern *regexp.Regexp
	Handler StepHandler
	// Source is a human-readable hint to the registration site (e.g. "builtin:http").
	Source string
}

// AmbiguousError is returned when more than one registered step matches.
type AmbiguousError struct {
	StepText string
	Matches  []*StepDef
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("ambiguous step %q: %d patterns matched", e.StepText, len(e.Matches))
}

// Registry holds all registered step definitions and provides matching.
type Registry struct {
	mu   sync.RWMutex
	defs []*StepDef
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a step pattern to the registry.
// pattern is a Go regexp anchored automatically at both ends.
func (r *Registry) Register(pattern string, handler StepHandler, source string) error {
	re, err := regexp.Compile(`^(?:` + pattern + `)$`)
	if err != nil {
		return fmt.Errorf("invalid step pattern %q: %w", pattern, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defs = append(r.defs, &StepDef{Pattern: re, Handler: handler, Source: source})
	return nil
}

// Match finds the single step definition that matches stepText.
// Returns ErrUndefined if no definition matches.
// Returns *AmbiguousError if more than one definition matches.
func (r *Registry) Match(stepText string) (*StepDef, []string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type candidate struct {
		def  *StepDef
		args []string
	}
	var hits []candidate
	for _, def := range r.defs {
		sub := def.Pattern.FindStringSubmatch(stepText)
		if sub != nil {
			hits = append(hits, candidate{def, sub[1:]})
		}
	}
	switch len(hits) {
	case 0:
		return nil, nil, ErrUndefined
	case 1:
		return hits[0].def, hits[0].args, nil
	default:
		ambig := make([]*StepDef, len(hits))
		for i, h := range hits {
			ambig[i] = h.def
		}
		return nil, nil, &AmbiguousError{StepText: stepText, Matches: ambig}
	}
}

// MatchStep matches a parsed Step against the registry.
// It is a convenience wrapper that passes step.Text.
func (r *Registry) MatchStep(step *parser.Step) (*StepDef, []string, error) {
	return r.Match(step.Text)
}

// StepRegistrar is the extension interface for registering custom step packages.
// Implementations should call registry.Register for each step they provide.
type StepRegistrar interface {
	Register(registry *Registry) error
}

// RegisterAll applies all registrars to the registry, returning the first error.
func RegisterAll(r *Registry, registrars ...StepRegistrar) error {
	for _, reg := range registrars {
		if err := reg.Register(r); err != nil {
			return err
		}
	}
	return nil
}
