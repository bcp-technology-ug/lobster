package steps

import (
	"net/http"

	"github.com/bcp-technology-ug/lobster/internal/parser"
)

// ScenarioContext is the per-scenario execution state threaded through every
// step handler in the scenario. A fresh ScenarioContext is created for each
// scenario before any steps are executed.
type ScenarioContext struct {
	// Variables holds scenario-scoped key-value substitution variables.
	Variables map[string]string

	// SuiteVars holds suite-scoped shared variables (read/write across scenarios).
	SuiteVars map[string]string

	// BaseURL is the HTTP base URL used by built-in HTTP steps.
	BaseURL string

	// DefaultHeaders are sent with every HTTP request unless overridden.
	DefaultHeaders map[string]string

	// HTTPClient is the HTTP client used by built-in HTTP steps.
	// Defaults to http.DefaultClient if nil.
	HTTPClient *http.Client

	// --- mutable HTTP state populated during step execution ---

	// LastResponse is the most recent HTTP response received.
	LastResponse *http.Response

	// LastBody is the raw response body bytes of LastResponse.
	LastBody []byte

	// LastRequest is the most recent HTTP request sent.
	LastRequest *http.Request

	// CurrentStep is set by the runner just before invoking the step handler.
	// Step handlers may inspect it for DataTable / DocString access.
	CurrentStep *parser.Step

	// SoftAssertMode collects assertion failures instead of stopping on the first.
	SoftAssertMode bool

	// assertionErrors accumulates failures in soft-assert mode.
	assertionErrors []error
}

// NewScenarioContext creates a ScenarioContext with sensible defaults.
func NewScenarioContext(baseURL string, defaultHeaders map[string]string, suiteVars map[string]string) *ScenarioContext {
	dh := make(map[string]string, len(defaultHeaders))
	for k, v := range defaultHeaders {
		dh[k] = v
	}
	sv := suiteVars
	if sv == nil {
		sv = make(map[string]string)
	}
	return &ScenarioContext{
		Variables:      make(map[string]string),
		SuiteVars:      sv,
		BaseURL:        baseURL,
		DefaultHeaders: dh,
		HTTPClient:     http.DefaultClient,
	}
}

// AddAssertionError records a soft assertion failure. In hard-fail mode it is
// unused; the runner is responsible for checking SoftAssertMode.
func (c *ScenarioContext) AddAssertionError(err error) {
	c.assertionErrors = append(c.assertionErrors, err)
}

// AssertionErrors returns all collected soft-assert failures.
func (c *ScenarioContext) AssertionErrors() []error {
	return c.assertionErrors
}

// HasAssertionFailures returns true when at least one soft-assert failed.
func (c *ScenarioContext) HasAssertionFailures() bool {
	return len(c.assertionErrors) > 0
}
