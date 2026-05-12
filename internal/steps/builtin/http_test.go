package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

func newScenarioCtx(baseURL string) *steps.ScenarioContext {
	return steps.NewScenarioContext(baseURL, nil, nil)
}

func TestHTTP_GET_statusOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "hello")
	}))
	t.Cleanup(srv.Close)

	ctx := newScenarioCtx(srv.URL)
	ctx.HTTPClient = srv.Client()

	if err := stepSendRequest(ctx, "GET", "/"); err != nil {
		t.Fatalf("stepSendRequest: %v", err)
	}
	if err := stepAssertStatusCode(ctx, "200"); err != nil {
		t.Fatalf("status assertion failed: %v", err)
	}
}

func TestHTTP_POST_withBody(t *testing.T) {
	t.Parallel()
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = buf
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	ctx := newScenarioCtx(srv.URL)
	ctx.HTTPClient = srv.Client()

	if err := doRequest(ctx, "POST", "/items", []byte(`{"name":"test"}`), "application/json"); err != nil {
		t.Fatalf("doRequest: %v", err)
	}
	if err := stepAssertStatusCode(ctx, "201"); err != nil {
		t.Fatalf("status assertion failed: %v", err)
	}
	if len(gotBody) == 0 {
		t.Error("expected server to receive body")
	}
}

func TestHTTP_assertBodyContains(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "welcome to lobster")
	}))
	t.Cleanup(srv.Close)

	ctx := newScenarioCtx(srv.URL)
	ctx.HTTPClient = srv.Client()

	if err := stepSendRequest(ctx, "GET", "/"); err != nil {
		t.Fatalf("stepSendRequest: %v", err)
	}
	if err := stepAssertBodyContains(ctx, "lobster"); err != nil {
		t.Fatalf("body contains failed: %v", err)
	}
	if err := stepAssertBodyNotContains(ctx, "nonexistent"); err != nil {
		t.Fatalf("body not-contains failed: %v", err)
	}
}

func TestHTTP_assertBodyIsJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	t.Cleanup(srv.Close)

	ctx := newScenarioCtx(srv.URL)
	ctx.HTTPClient = srv.Client()

	if err := stepSendRequest(ctx, "GET", "/"); err != nil {
		t.Fatalf("stepSendRequest: %v", err)
	}
	if err := stepAssertBodyIsJSON(ctx); err != nil {
		t.Fatalf("assertBodyIsJSON failed: %v", err)
	}
}

func TestHTTP_assertResponseHeader(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Correlation-ID", "test-123")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx := newScenarioCtx(srv.URL)
	ctx.HTTPClient = srv.Client()

	if err := stepSendRequest(ctx, "GET", "/"); err != nil {
		t.Fatalf("stepSendRequest: %v", err)
	}
	if err := stepAssertResponseHeader(ctx, "X-Correlation-ID", "test-123"); err != nil {
		t.Fatalf("assertResponseHeader failed: %v", err)
	}
}

func TestHTTP_storeBodyInVar(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "payload-value")
	}))
	t.Cleanup(srv.Close)

	ctx := newScenarioCtx(srv.URL)
	ctx.HTTPClient = srv.Client()

	if err := stepSendRequest(ctx, "GET", "/"); err != nil {
		t.Fatalf("stepSendRequest: %v", err)
	}
	if err := stepStoreBodyInVar(ctx, "myVar"); err != nil {
		t.Fatalf("stepStoreBodyInVar: %v", err)
	}
	if got := ctx.Variables["myVar"]; got != "payload-value" {
		t.Errorf("variable: got %q want %q", got, "payload-value")
	}
}

func TestHTTP_statusCodeNotMatch_returnsError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	ctx := newScenarioCtx(srv.URL)
	ctx.HTTPClient = srv.Client()

	if err := stepSendRequest(ctx, "GET", "/"); err != nil {
		t.Fatalf("stepSendRequest: %v", err)
	}
	if err := stepAssertStatusCode(ctx, "200"); err == nil {
		t.Error("expected assertion to fail for status 404 vs 200")
	}
}

func TestBuildURL_basic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		base, path, want string
	}{
		{"http://localhost:8080", "/api", "http://localhost:8080/api"},
		{"http://localhost:8080/", "/api", "http://localhost:8080/api"},
		{"http://localhost:8080", "api", "http://localhost:8080/api"},
		{"", "/relative", "/relative"},
	}
	for _, tc := range tests {
		got := buildURL(tc.base, tc.path)
		if got != tc.want {
			t.Errorf("buildURL(%q, %q) = %q want %q", tc.base, tc.path, got, tc.want)
		}
	}
}
