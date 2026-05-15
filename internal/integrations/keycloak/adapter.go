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

// GetUserTokenForRealm obtains a user token for an arbitrary realm using the
// password grant flow. clientID defaults to "account" when empty.
func (a *Adapter) GetUserTokenForRealm(ctx context.Context, realm, username, password, clientID string) (string, error) {
	if clientID == "" {
		clientID = "account"
	}
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		strings.TrimRight(a.cfg.BaseURL, "/"), url.PathEscape(realm))

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

// EnsureRealmExists ensures the named realm exists, creating it when absent.
func (a *Adapter) EnsureRealmExists(ctx context.Context, realmName string) error {
	token, err := a.GetToken(ctx)
	if err != nil {
		return err
	}
	realmURL := fmt.Sprintf("%s/admin/realms/%s",
		strings.TrimRight(a.cfg.BaseURL, "/"), url.PathEscape(realmName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, realmURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("check realm %q: %w", realmName, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck,gosec

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("check realm %q: HTTP %d", realmName, resp.StatusCode)
	}

	// Create realm.
	payload := map[string]interface{}{"realm": realmName, "enabled": true}
	body, _ := json.Marshal(payload)
	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/admin/realms", strings.TrimRight(a.cfg.BaseURL, "/")),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := a.client.Do(createReq)
	if err != nil {
		return fmt.Errorf("create realm %q: %w", realmName, err)
	}
	defer createResp.Body.Close()
	io.Copy(io.Discard, createResp.Body) //nolint:errcheck,gosec

	if createResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create realm %q: HTTP %d", realmName, createResp.StatusCode)
	}
	return nil
}

// EnsureUserInRealm creates a user in the specified realm when absent and
// sets their password. The user's ID is returned. If the user already exists
// only the ID is returned without modifying the existing record.
func (a *Adapter) EnsureUserInRealm(ctx context.Context, realmName, username, password string) (string, error) {
	token, err := a.GetToken(ctx)
	if err != nil {
		return "", err
	}

	// Check whether the user already exists.
	searchURL := fmt.Sprintf("%s/admin/realms/%s/users?username=%s&exact=true",
		strings.TrimRight(a.cfg.BaseURL, "/"), url.PathEscape(realmName), url.QueryEscape(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("search user %q: %w", username, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("search user %q: HTTP %d: %s", username, resp.StatusCode, body)
	}

	var users []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return "", fmt.Errorf("decode users: %w", err)
	}
	if len(users) > 0 {
		return users[0].ID, nil
	}

	// Create user.
	userPayload := map[string]interface{}{
		"username": username,
		"enabled":  true,
	}
	userBody, _ := json.Marshal(userPayload)
	createURL := fmt.Sprintf("%s/admin/realms/%s/users",
		strings.TrimRight(a.cfg.BaseURL, "/"), url.PathEscape(realmName))
	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(userBody))
	if err != nil {
		return "", err
	}
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := a.client.Do(createReq)
	if err != nil {
		return "", fmt.Errorf("create user %q: %w", username, err)
	}
	defer createResp.Body.Close()
	io.Copy(io.Discard, createResp.Body) //nolint:errcheck,gosec

	if createResp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create user %q: HTTP %d", username, createResp.StatusCode)
	}

	// Retrieve user ID from Location header.
	location := createResp.Header.Get("Location")
	parts := strings.Split(strings.TrimRight(location, "/"), "/")
	userID := parts[len(parts)-1]

	// Set password.
	if password != "" {
		if err := a.setUserPassword(ctx, token, realmName, userID, password); err != nil {
			return userID, err
		}
	}
	return userID, nil
}

func (a *Adapter) setUserPassword(ctx context.Context, token, realmName, userID, password string) error {
	cred := map[string]interface{}{
		"type":      "password",
		"value":     password,
		"temporary": false,
	}
	body, _ := json.Marshal(cred)
	pwURL := fmt.Sprintf("%s/admin/realms/%s/users/%s/reset-password",
		strings.TrimRight(a.cfg.BaseURL, "/"), url.PathEscape(realmName), url.PathEscape(userID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, pwURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck,gosec

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("set password: HTTP %d", resp.StatusCode)
	}
	return nil
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
	io.Copy(io.Discard, resp.Body) //nolint:errcheck,gosec

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
	io.Copy(io.Discard, resp.Body) //nolint:errcheck,gosec

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create realm: HTTP %d", resp.StatusCode)
	}
	return nil
}
