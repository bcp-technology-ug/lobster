package ui

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// ErrFormAborted is returned when the user exits a Huh form before submitting.
var ErrFormAborted = huh.ErrUserAborted

// IsFormAborted reports whether err indicates the user cancelled a form.
func IsFormAborted(err error) bool {
	return errors.Is(err, huh.ErrUserAborted)
}

// InitFields holds the values collected by the init form.
// All fields have sensible defaults applied before the form runs.
type InitFields struct {
	Project   string
	Workspace string
	Features  string
	Compose   string
}

// NewInitForm returns a Huh form that populates dst.
// Call form.Run() to execute it. dst must be non-nil.
func NewInitForm(dst *InitFields) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Description("Identifier used across run history and reports.").
				Placeholder("my-project").
				Value(&dst.Project).
				Validate(func(s string) error {
					if s == "" {
						return errorf("project name is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Default workspace").
				Description("Leave blank to use the root workspace.").
				Placeholder("(root)").
				Value(&dst.Workspace),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Feature file glob").
				Description("Glob pattern pointing to your .feature files.").
				Placeholder("features/**/*.feature").
				Value(&dst.Features),

			huh.NewInput().
				Title("Docker Compose file").
				Description("Leave blank to configure manually later.").
				Placeholder("docker-compose.yaml").
				Value(&dst.Compose),
		),
	)
}

// NewConfirmForm returns a single yes/no Huh form.
// dst receives the user's choice. Call form.Run() to execute.
func NewConfirmForm(question string, dst *bool) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(question).
				Value(dst),
		),
	)
}

// errorf is a tiny helper so we avoid importing fmt in validate closures.
func errorf(msg string) error {
	return &formError{msg: msg}
}

type formError struct{ msg string }

func (e *formError) Error() string { return e.msg }
