// Package keycloak provides a Keycloak integration adapter for Lobster.
// It automates realm, client, user, and role provisioning so test suites
// have deterministic identity state before each run.
package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config holds the Keycloak connection and provisioning settings.
type Config struct {
	// BaseURL is the Keycloak server URL, e.g. "http://keycloak:8080".
	BaseURL string

	// AdminUser is the Keycloak admin username.
	AdminUser string

	// AdminPassword is the Keycloak admin password (loaded from env at
	// call-site; never stored in config files).
	AdminPassword string

	// Realm is the Keycloak realm to manage.
	Realm string

	// ResetBetweenScenarios triggers Reset on each scenario boundary.
	ResetBetweenScenarios bool

	// HTTPTimeout is the per-request timeout. Defaults to 30s.
	HTTPTimeout time.Duration
}

// Adapter implements integrations.Adapter for Keycloak.
type Adapter struct {
	id     string
	cfg    Config
	client *http.Client

	mu    sync.Mutex
	token string
	exp   time.Time
}

// New creates a Keycloak Adapter. id must be unique within the registry.
func New(id string, cfg Config) *Adapter {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Adapter{
		id:     id,
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

// ID implements integrations.Adapter.
func (a *Adapter) ID() string { return a.id }

// Kind implements integrations.Adapter.
func (a *Adapter) Kind() string { return "keycloak" }

// Setup ensures the configured realm exists. It obtains an admin token,
// creates the realm if absent, and provisions any required base configuration.
func (a *Adapter) Setup(ctx context.Context) error {
	if err := a.refreshToken(ctx); err != nil {
		return fmt.Errorf("keycloak auth: %w", err)
	}
	if err := a.ensureRealm(ctx); err != nil {
		return fmt.Errorf("ensure realm %q: %w", a.cfg.Realm, err)
	}
	return nil
}

// Reset restores predictable realm state between scenarios.
// In v0.1, Reset re-authenticates (token may have expired) and is a no-op
// beyond that. Full user/role reset can be added per-project via sub-classing
// the adapter with additional Reset logic.
func (a *Adapter) Reset(ctx context.Context) error {
	return a.refreshToken(ctx)
}

// Teardown is a no-op in v0.1. Realm destruction is opt-in to avoid
// accidental deletion of persistent environments.
func (a *Adapter) Teardown(_ context.Context) error {
	return nil
}

// GetToken returns a valid admin token, refreshing if necessary.
// Exported so step definitions can use it for test token acquisition.
func (a *Adapter) GetToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if time.Now().Before(a.exp) {
		return a.token, nil
	}
	if err := a.doRefreshToken(ctx); err != nil {
		return "", err
	}
	return a.token, nil
}

// GetUserToken obtains a user token using the password grant flow.
// Useful in step definitions to acquire per-user Bearer tokens.
func (a *Adapter) GetUserToken(ctx context.Context, username, password, clientID string) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		strings.TrimRight(a.cfg.BaseURL, "/"), url.PathEscape(a.cfg.Realm))

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", clientID)
	form.Set("username", username)
	form.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("get user token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get user token: HTTP %d: %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	return result.AccessToken, nil
}

// --- internal helpers ---

func (a *Adapter) refreshToken(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if time.Now().Before(a.exp) {
		return nil
	}
	return a.doRefreshToken(ctx)
}

// doRefreshToken must be called with a.mu held.
func (a *Adapter) doRefreshToken(ctx context.Context) error {
	tokenURL := fmt.Sprintf("%s/realms/master/protocol/openid-connect/token",
		strings.TrimRight(a.cfg.BaseURL, "/"))

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "admin-cli")
	form.Set("username", a.cfg.AdminUser)
	form.Set("password", a.cfg.AdminPassword)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("keycloak token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("keycloak token HTTP %d: %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode token: %w", err)
	}
	a.token = result.AccessToken
	// Subtract 10s from expiry as safety margin.
	a.exp = time.Now().Add(time.Duration(result.ExpiresIn-10) * time.Second)
	return nil
}

func (a *Adapter) ensureRealm(ctx context.Context) error {
	token, err := a.GetToken(ctx)
	if err != nil {
		return err
	}

	realmURL := fmt.Sprintf("%s/admin/realms/%s",
		strings.TrimRight(a.cfg.BaseURL, "/"), url.PathEscape(a.cfg.Realm))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, realmURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("check realm: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode == http.StatusOK {
		return nil // realm exists
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("check realm: HTTP %d", resp.StatusCode)
	}

	// Create realm.
	return a.createRealm(ctx, token)
}

func (a *Adapter) createRealm(ctx context.Context, token string) error {
	payload := map[string]interface{}{
		"realm":   a.cfg.Realm,
		"enabled": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	realmsURL := fmt.Sprintf("%s/admin/realms", strings.TrimRight(a.cfg.BaseURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, realmsURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("create realm: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create realm: HTTP %d", resp.StatusCode)
	}
	return nil
}
