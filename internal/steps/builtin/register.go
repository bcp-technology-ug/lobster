// Package builtin provides Lobster's built-in step library.
// Steps are registered into a *steps.Registry via Register().
// Lifecycle hooks are registered into a *steps.HookRegistry via RegisterHooks().
package builtin

import (
	"os"

	"github.com/bcp-technology-ug/lobster/internal/steps"
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
	if err := registerShellSteps(r); err != nil {
		return err
	}
	if err := registerFSSteps(r); err != nil {
		return err
	}
	return nil
}

// RegisterHooks wires built-in lifecycle hooks into h.
//
// Currently registered hooks:
//   - AfterScenario: if the scenario used "I am in a new temporary directory",
//     restores the original working directory and removes the temp dir.
func RegisterHooks(h *steps.HookRegistry) {
	h.AfterScenario(func(sc *steps.ScenarioContext) error {
		tmp := sc.Variables[varTmpDir]
		if tmp == "" {
			return nil
		}
		orig := sc.Variables[varWorkDir]
		if orig != "" {
			_ = os.Chdir(orig)
		}
		return os.RemoveAll(tmp)
	})
}
