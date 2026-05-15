package steps

import "strings"

// HumanizePattern converts a raw regexp step pattern into a human-readable
// representation suitable for documentation and agent context windows.
//
// Transformations applied (in order):
//   - `(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)` → `GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS`
//   - `((?:[^"\\]|\\.)*)` (escaped-string capture) → `<string>`
//   - `([^"]+)` (unquoted-string capture) → `<string>`
//   - `(\d+)s?` (seconds) → `<n>s`
//   - `(\d+)` (integer) → `<n>`
//
// The result is still valid for human reading but NOT for regexp matching.
func HumanizePattern(raw string) string {
	r := raw

	// HTTP method alternation — remove outer parens.
	r = strings.ReplaceAll(r, `(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)`, `GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS`)

	// Escaped-string capture group (used in shell commands).
	r = strings.ReplaceAll(r, `((?:[^"\\]|\\.)*)`, `<string>`)

	// Standard quoted-string capture.
	r = strings.ReplaceAll(r, `([^"]+)`, `<string>`)

	// Integer with optional trailing 's' (timeouts, waits).
	r = strings.ReplaceAll(r, `(\d+)s?`, `<n>s`)

	// Plain integer capture.
	r = strings.ReplaceAll(r, `(\d+)`, `<n>`)

	return r
}
