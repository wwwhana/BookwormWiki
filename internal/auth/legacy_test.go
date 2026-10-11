package auth

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Credentials issued while the project was called Stash keep working after
// the rename to BookwormWiki.
func TestStashEraCredentialsStillWork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := openAuthTestSchema(t, ctx, nil)
	p := &Provider{config: Config{Mode: "token", APISecret: testSigningSecret}, tokenPool: pool}

	// An API token minted before the rename, stored by its digest.
	legacy := legacyAPITokenPrefix + "issued-before-the-rename"
	hash := sha256.Sum256([]byte(legacy))
	if _, err := pool.Exec(ctx, `INSERT INTO auth_tokens (subject, name, token_hash) VALUES ('codex', 'old laptop', $1)`, hash[:]); err != nil {
		t.Fatal(err)
	}
	mcp := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	mcp.Header.Set("Authorization", "Bearer "+legacy)
	if got, err := p.VerifyMCPRequest(mcp); err != nil || got != "codex" {
		t.Fatalf("legacy API token over MCP = %q, %v", got, err)
	}
	if got, err := p.VerifyBearerToken(ctx, legacy); err != nil || got != "codex" {
		t.Fatalf("legacy API token over STDIO = %q, %v", got, err)
	}
	// New tokens use the new prefix.
	fresh, _, err := p.IssueAPIToken(ctx, "codex", "new laptop", 0)
	if err != nil || !strings.HasPrefix(fresh, "bookwormwiki_api_") {
		t.Fatalf("new token = %q, %v", fresh, err)
	}

	// A browser session cookie set before the rename.
	payload, err := signSessionToken("alice", testSigningSecret, time.Now().Add(time.Hour), true)
	if err != nil {
		t.Fatal(err)
	}
	legacySession := legacySessionTokenPrefix + strings.TrimPrefix(payload, sessionTokenPrefix)
	console := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
	console.AddCookie(&http.Cookie{Name: legacySessionCookieName, Value: legacySession})
	if got, err := p.VerifyRequest(console); err != nil || got != "alice" {
		t.Fatalf("legacy session cookie = %q, %v", got, err)
	}
	// The first request moves the browser to the new cookie name.
	rec := httptest.NewRecorder()
	p.RenewSession(rec, console)
	var renewed, cleared bool
	for _, cookie := range rec.Result().Cookies() {
		switch {
		case cookie.Name == sessionCookieName && strings.HasPrefix(cookie.Value, sessionTokenPrefix):
			renewed = true
		case cookie.Name == legacySessionCookieName && cookie.MaxAge < 0:
			cleared = true
		}
	}
	if !renewed || !cleared {
		t.Fatalf("legacy session was not migrated: %#v", rec.Result().Cookies())
	}
	// A non-renewable session from before the rename moves too, keeping its expiry.
	fixedExpiry := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	fixed, err := signSessionToken("bob", testSigningSecret, fixedExpiry, false)
	if err != nil {
		t.Fatal(err)
	}
	fixedLegacy := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
	fixedLegacy.AddCookie(&http.Cookie{Name: legacySessionCookieName, Value: legacySessionTokenPrefix + strings.TrimPrefix(fixed, sessionTokenPrefix)})
	moved := httptest.NewRecorder()
	p.RenewSession(moved, fixedLegacy)
	var movedCookie *http.Cookie
	for _, cookie := range moved.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			movedCookie = cookie
		}
	}
	if movedCookie == nil {
		t.Fatalf("non-renewable legacy session was not moved: %#v", moved.Result().Cookies())
	}
	if subject, expiresAt, renewable, err := parseSessionClaims(movedCookie.Value, testSigningSecret); err != nil || subject != "bob" || renewable || !expiresAt.Equal(fixedExpiry) {
		t.Fatalf("moved session = %q %v renewable=%v %v", subject, expiresAt, renewable, err)
	}
	// With a current session present, the legacy cookie is only dropped.
	both := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
	both.AddCookie(&http.Cookie{Name: sessionCookieName, Value: payload})
	both.AddCookie(&http.Cookie{Name: legacySessionCookieName, Value: legacySession})
	dropped := httptest.NewRecorder()
	p.RenewSession(dropped, both)
	if cookies := dropped.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != legacySessionCookieName || cookies[0].MaxAge >= 0 {
		t.Fatalf("legacy cookie next to a current session = %#v", cookies)
	}
	if got, err := p.VerifyRequest(both); err != nil || got != "alice" {
		t.Fatalf("current session must win over the legacy cookie: %q, %v", got, err)
	}

	// Logging out clears both names.
	logout := httptest.NewRecorder()
	p.HandleLogout(logout, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	expired := map[string]bool{}
	for _, cookie := range logout.Result().Cookies() {
		if cookie.MaxAge < 0 {
			expired[cookie.Name] = true
		}
	}
	if !expired[sessionCookieName] || !expired[legacySessionCookieName] {
		t.Fatalf("logout cookies = %#v", logout.Result().Cookies())
	}
	// A legacy session token is still refused as a bearer credential.
	bearer := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	bearer.Header.Set("Authorization", "Bearer "+legacySession)
	if _, err := p.VerifyMCPRequest(bearer); err == nil {
		t.Fatal("a session token was accepted as an MCP bearer credential")
	}
}
