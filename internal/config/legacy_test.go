package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyStashEnvironmentStillConfiguresTheServer(t *testing.T) {
	for key, value := range map[string]string{
		"STASH_POSTGRES_DSN":    "postgres://legacy/stash",
		"STASH_MAX_RESULT_SIZE": "4242",
		"STASH_CONTEXT_TTL":     "1h",
		"STASH_HTTP_ADDR":       "127.0.0.1:8080",
		"STASH_LOG_LEVEL":       "warn",
		"STASH_LOG_FORMAT":      "text",
		// The new name wins when both are set.
		"BOOKWORMWIKI_LOG_LEVEL": "debug",
	} {
		t.Setenv(key, value)
	}
	t.Cleanup(func() {
		for _, name := range []string{"BOOKWORMWIKI_POSTGRES_DSN", "BOOKWORMWIKI_MAX_RESULT_SIZE", "BOOKWORMWIKI_CONTEXT_TTL", "BOOKWORMWIKI_HTTP_ADDR", "BOOKWORMWIKI_LOG_FORMAT", "BOOKWORMWIKI_AUTH_MODE", "BOOKWORMWIKI_VECTOR_DIM"} {
			_ = os.Unsetenv(name)
		}
	})
	dir := t.TempDir()
	envFile := filepath.Join(dir, "stash.env")
	// A .env file written for Stash keeps working too.
	if err := os.WriteFile(envFile, []byte("STASH_AUTH_MODE=none\nSTASH_VECTOR_DIM=768\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := NewFromFile(envFile)
	if err != nil {
		t.Fatalf("load legacy config: %v", err)
	}
	if cfg.StoreDSN != "postgres://legacy/stash" || cfg.MaxResultSize != 4242 || cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("legacy process variables were not applied: dsn=%q size=%d addr=%q", cfg.StoreDSN, cfg.MaxResultSize, cfg.HTTPAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("BOOKWORMWIKI_LOG_LEVEL must win over STASH_LOG_LEVEL, got %q", cfg.LogLevel)
	}
	if cfg.AuthMode != "none" || cfg.VectorDim != 768 {
		t.Fatalf("legacy .env entries were not applied: mode=%q dim=%d", cfg.AuthMode, cfg.VectorDim)
	}
	names := map[string]bool{}
	for _, name := range LegacyEnvironmentNames() {
		names[name] = true
	}
	for _, want := range []string{"STASH_POSTGRES_DSN", "STASH_AUTH_MODE", "STASH_VECTOR_DIM"} {
		if !names[want] {
			t.Fatalf("%s missing from the deprecation list %v", want, LegacyEnvironmentNames())
		}
	}
	if names["STASH_LOG_LEVEL"] {
		t.Fatal("a legacy variable shadowed by its new name must not be reported as used")
	}
}

func TestLegacyConfigFileVariable(t *testing.T) {
	t.Setenv("STASHCONFIG", "/etc/stash/.env")
	t.Cleanup(func() { _ = os.Unsetenv("BOOKWORMWIKICONFIG") })
	_ = os.Unsetenv("BOOKWORMWIKICONFIG")
	ApplyLegacyEnvironment()
	if got := os.Getenv("BOOKWORMWIKICONFIG"); got != "/etc/stash/.env" {
		t.Fatalf("BOOKWORMWIKICONFIG = %q", got)
	}
}
