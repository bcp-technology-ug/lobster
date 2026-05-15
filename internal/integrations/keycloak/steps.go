package keycloak

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

// RegisterSteps registers Keycloak-specific Gherkin step definitions into reg.
// The handlers close over the adapter so they can call Keycloak admin APIs
// at scenario runtime. Registered patterns:
//
//   - `keycloak realm "REALM" exists`
//   - `user "USERNAME" exists in realm "REALM"`
//   - `user "USERNAME" exists in realm "REALM" with password "PASSWORD"`
//   - `I request an access token for user "USERNAME" in realm "REALM"`
//   - `the token should contain role "ROLE"`
func (a *Adapter) RegisterSteps(reg *steps.Registry) error {
	type registration struct {
		pattern string
		handler steps.StepHandler
	}
	defs := []registration{
		{
			pattern: `keycloak realm "([^"]+)" exists`,
			handler: a.stepEnsureRealm,
		},
		{
			pattern: `user "([^"]+)" exists in realm "([^"]+)"`,
			handler: a.stepEnsureUser,
		},
		{
			pattern: `user "([^"]+)" exists in realm "([^"]+)" with password "([^"]+)"`,
			handler: a.stepEnsureUserWithPassword,
		},
		{
			pattern: `I request an access token for user "([^"]+)" in realm "([^"]+)"`,
			handler: a.stepRequestAccessToken,
		},
		{
			pattern: `the token should contain role "([^"]+)"`,
			handler: stepTokenContainsRole,
		},
	}
	for _, d := range defs {
		if err := reg.Register(d.pattern, d.handler, "keycloak"); err != nil {
			return fmt.Errorf("register keycloak step %q: %w", d.pattern, err)
		}
	}
	return nil
}

// stepEnsureRealm handles: `keycloak realm "REALM" exists`
func (a *Adapter) stepEnsureRealm(_ *steps.ScenarioContext, args ...string) error {
	realmName := args[0]
	return a.EnsureRealmExists(context.Background(), realmName)
}

// stepEnsureUser handles: `user "USERNAME" exists in realm "REALM"`
// Creates the user with a default password of "lobster-test-{username}" and
// stores it in ctx.Variables as "{username}_password".
func (a *Adapter) stepEnsureUser(ctx *steps.ScenarioContext, args ...string) error {
	username := args[0]
	realmName := args[1]
	password := "lobster-test-" + username
	if _, err := a.EnsureUserInRealm(context.Background(), realmName, username, password); err != nil {
		return err
	}
	ctx.Variables[username+"_password"] = password
	return nil
}

// stepEnsureUserWithPassword handles: `user "USERNAME" exists in realm "REALM" with password "PASSWORD"`
// Stores the password in ctx.Variables as "{username}_password".
func (a *Adapter) stepEnsureUserWithPassword(ctx *steps.ScenarioContext, args ...string) error {
	username := args[0]
	realmName := args[1]
	password := args[2]
	if _, err := a.EnsureUserInRealm(context.Background(), realmName, username, password); err != nil {
		return err
	}
	ctx.Variables[username+"_password"] = password
	return nil
}

// stepRequestAccessToken handles: `I request an access token for user "USERNAME" in realm "REALM"`
// Looks up the password from ctx.Variables["{username}_password"] and stores
// the resulting access token in ctx.Variables["keycloak_access_token"].
func (a *Adapter) stepRequestAccessToken(ctx *steps.ScenarioContext, args ...string) error {
	username := args[0]
	realmName := args[1]

	password, ok := ctx.Variables[username+"_password"]
	if !ok {
		return fmt.Errorf("no password stored for user %q; ensure the user exists step ran first", username)
	}

	token, err := a.GetUserTokenForRealm(context.Background(), realmName, username, password, "")
	if err != nil {
		return fmt.Errorf("get access token for user %q: %w", username, err)
	}
	ctx.Variables["keycloak_access_token"] = token
	return nil
}

// stepTokenContainsRole handles: `the token should contain role "ROLE"`
// Reads ctx.Variables["keycloak_access_token"] and checks Keycloak's
// realm_access.roles claim for the named role.
func stepTokenContainsRole(ctx *steps.ScenarioContext, args ...string) error {
	role := args[0]
	rawToken, ok := ctx.Variables["keycloak_access_token"]
	if !ok || rawToken == "" {
		return fmt.Errorf("no access token available; run the token request step first")
	}
	return checkJWTContainsRole(rawToken, role)
}

// checkJWTContainsRole decodes a JWT payload (without signature verification —
// suitable for test assertions only) and checks realm_access.roles.
func checkJWTContainsRole(rawToken, role string) error {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return fmt.Errorf("malformed JWT: expected 3 parts, got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode JWT payload: %w", err)
	}

	var claims struct {
		RealmAccess struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("parse JWT claims: %w", err)
	}

	for _, r := range claims.RealmAccess.Roles {
		if r == role {
			return nil
		}
	}
	return fmt.Errorf("token does not contain realm role %q; present roles: %v", role, claims.RealmAccess.Roles)
}
