// Package builtin provides Lobster's built-in step library.
// Steps are registered into a *steps.Registry via Register().
package builtin

import (
	"github.com/bcp-technology/lobster/internal/steps"
)

const srcHTTP = "builtin:http"
const srcService = "builtin:service"

// Register adds all built-in steps to registry.
func Register(r *steps.Registry) error {
	if err := registerHTTPSteps(r); err != nil {
		return err
	}
	if err := registerServiceSteps(r); err != nil {
		return err
	}
	return nil
}
