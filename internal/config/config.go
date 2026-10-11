package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"github.com/wwwhana/bookwormwiki/internal/secrets"
)

type Config struct {
	// Store (PostgreSQL only)
	StoreDSN      string `env:"BOOKWORMWIKI_POSTGRES_DSN,required"`
	VectorDim     int    `env:"BOOKWORMWIKI_VECTOR_DIM" envDefault:"0"`
	MaxResultSize int    `env:"BOOKWORMWIKI_MAX_RESULT_SIZE,required"`

	// OpenAI-compatible environment provider. All of these are optional: a
	// deployment can register providers in the database instead, and a
	// server without any embedding provider still stores memories and
	// searches them with trigram matching.
	OpenAIAPIKey              string        `env:"BOOKWORMWIKI_OPENAI_API_KEY" envDefault:""`
	OpenAIBaseURL             string        `env:"BOOKWORMWIKI_OPENAI_BASE_URL" envDefault:""`
	EmbeddingModel            string        `env:"BOOKWORMWIKI_EMBEDDING_MODEL" envDefault:""`
	ReasonerModel             string        `env:"BOOKWORMWIKI_REASONER_MODEL" envDefault:""`
	EmbeddingRetryInterval    time.Duration `env:"BOOKWORMWIKI_EMBEDDING_RETRY_INTERVAL" envDefault:"1m"`
	EmbeddingRetryMaxInterval time.Duration `env:"BOOKWORMWIKI_EMBEDDING_RETRY_MAX_INTERVAL" envDefault:"1h"`
	EmbeddingRetryBatchSize   int           `env:"BOOKWORMWIKI_EMBEDDING_RETRY_BATCH_SIZE" envDefault:"100"`
	// OpenAI-compatible providers are allowed to be remote or local. Keep one
	// request from holding an MCP call (or the background retry worker) forever.
	OpenAIRequestTimeout time.Duration `env:"BOOKWORMWIKI_OPENAI_REQUEST_TIMEOUT" envDefault:"2m"`
	// Model input limits are provider/model-specific. MCP cannot discover them
	// or the caller's remaining context, so deployments may set these values to
	// make consolidation and long-memory embedding split before a request fails.
	ReasonerContextTokens  int `env:"BOOKWORMWIKI_REASONER_CONTEXT_TOKENS" envDefault:"0"`
	ReasonerReservedTokens int `env:"BOOKWORMWIKI_REASONER_RESERVED_TOKENS" envDefault:"4096"`
	EmbeddingContextTokens int `env:"BOOKWORMWIKI_EMBEDDING_CONTEXT_TOKENS" envDefault:"0"`

	// EmbeddingCache keeps computed vectors in embedding_cache so identical
	// text is not sent to the provider twice. It is an optimization only.
	EmbeddingCache bool `env:"BOOKWORMWIKI_EMBEDDING_CACHE" envDefault:"true"`

	// Secrets seal provider API keys stored in the database. The active key
	// seals new values; previous keys (comma-separated) only open old ones so
	// rotation never needs every credential re-entered at once.
	SecretsKey         string `env:"BOOKWORMWIKI_SECRETS_KEY" envDefault:""`
	SecretsKeyPrevious string `env:"BOOKWORMWIKI_SECRETS_KEY_PREVIOUS" envDefault:""`

	// Memory
	ContextTTL time.Duration `env:"BOOKWORMWIKI_CONTEXT_TTL,required"`

	// Server
	HTTPAddr            string `env:"BOOKWORMWIKI_HTTP_ADDR,required"`
	LogLevel            string `env:"BOOKWORMWIKI_LOG_LEVEL,required"`
	LogFormat           string `env:"BOOKWORMWIKI_LOG_FORMAT,required"`
	MCPMaxResponseBytes int    `env:"BOOKWORMWIKI_MCP_MAX_RESPONSE_BYTES" envDefault:"32768"`
	// Tool handlers use a separate bound so database-only calls cannot remain
	// queued behind a stalled provider request until the client's timeout.
	MCPToolTimeout time.Duration `env:"BOOKWORMWIKI_MCP_TOOL_TIMEOUT" envDefault:"2m"`

	// Authentication
	AuthMode         string        `env:"BOOKWORMWIKI_AUTH_MODE" envDefault:"none"`
	AuthIssuer       string        `env:"BOOKWORMWIKI_AUTH_ISSUER" envDefault:""`
	AuthClientID     string        `env:"BOOKWORMWIKI_AUTH_CLIENT_ID" envDefault:""`
	AuthClientSecret string        `env:"BOOKWORMWIKI_AUTH_CLIENT_SECRET" envDefault:""`
	AuthRedirectURL  string        `env:"BOOKWORMWIKI_AUTH_REDIRECT_URL" envDefault:""`
	AuthAPISecret    string        `env:"BOOKWORMWIKI_AUTH_API_SECRET" envDefault:""`
	AuthCookieSecure bool          `env:"BOOKWORMWIKI_AUTH_COOKIE_SECURE" envDefault:"true"`
	AuthTokenTTL     time.Duration `env:"BOOKWORMWIKI_AUTH_TOKEN_TTL" envDefault:"720h"`
	AuthSessionTTL   time.Duration `env:"BOOKWORMWIKI_AUTH_SESSION_TTL" envDefault:"720h"`
	AuthStdioToken   string        `env:"BOOKWORMWIKI_AUTH_STDIO_TOKEN" envDefault:""`
	// Admin maintenance accepts either an authenticated OIDC subject listed
	// here or the separate static token below. Keep this independent from the
	// MCP API secret so a maintenance credential cannot sign user sessions.
	AdminSubjects string `env:"BOOKWORMWIKI_ADMIN_SUBJECTS" envDefault:""`
	AdminToken    string `env:"BOOKWORMWIKI_ADMIN_TOKEN" envDefault:""`
	// The first local administrator is created from these at startup when
	// the account does not exist yet; later password changes are kept.
	AdminUser     string `env:"BOOKWORMWIKI_ADMIN_USER" envDefault:""`
	AdminPassword string `env:"BOOKWORMWIKI_ADMIN_PASSWORD" envDefault:""`
	// AuthTrustedNetwork lets BOOKWORMWIKI_AUTH_MODE=none listen beyond loopback. It
	// is an explicit statement that everyone on the network may use BookwormWiki.
	AuthTrustedNetwork bool `env:"BOOKWORMWIKI_AUTH_TRUSTED_NETWORK" envDefault:"false"`

	// OAuth-prefixed aliases make the profile explicit while preserving the
	// original BOOKWORMWIKI_AUTH_* names used by existing deployments.
	AuthOAuthIssuer          string        `env:"BOOKWORMWIKI_AUTH_OAUTH_ISSUER" envDefault:""`
	AuthOAuthClientID        string        `env:"BOOKWORMWIKI_AUTH_OAUTH_CLIENT_ID" envDefault:""`
	AuthOAuthClientSecret    string        `env:"BOOKWORMWIKI_AUTH_OAUTH_CLIENT_SECRET" envDefault:""`
	AuthOAuthRedirectURL     string        `env:"BOOKWORMWIKI_AUTH_OAUTH_REDIRECT_URL" envDefault:""`
	AuthOAuthAPISecret       string        `env:"BOOKWORMWIKI_AUTH_OAUTH_API_SECRET" envDefault:""`
	AuthOAuthCookieSecureRaw string        `env:"BOOKWORMWIKI_AUTH_OAUTH_COOKIE_SECURE" envDefault:""`
	AuthOAuthTokenTTL        time.Duration `env:"BOOKWORMWIKI_AUTH_OAUTH_TOKEN_TTL" envDefault:"0s"`
	AuthOAuthStdioToken      string        `env:"BOOKWORMWIKI_AUTH_OAUTH_STDIO_TOKEN" envDefault:""`

	// Consolidation
	ConsolidationBatchSize           int     `env:"BOOKWORMWIKI_CONSOLIDATION_BATCH_SIZE" envDefault:"100"`
	ConsolidationSimilarityThreshold float64 `env:"BOOKWORMWIKI_CONSOLIDATION_SIMILARITY_THRESHOLD" envDefault:"0.85"`
	ConsolidationDedupThreshold      float64 `env:"BOOKWORMWIKI_CONSOLIDATION_DEDUP_THRESHOLD" envDefault:"0.85"`
	ConsolidationWindow              string  `env:"BOOKWORMWIKI_CONSOLIDATION_WINDOW" envDefault:"168h"`
	DecayFactor                      float64 `env:"BOOKWORMWIKI_DECAY_FACTOR" envDefault:"0.95"`
	ExpiryThreshold                  float32 `env:"BOOKWORMWIKI_EXPIRY_THRESHOLD" envDefault:"0.1"`
	HypothesisAutoConfirmThreshold   float32 `env:"BOOKWORMWIKI_HYPOTHESIS_AUTO_CONFIRM_THRESHOLD" envDefault:"0.9"`
	HypothesisAutoRejectThreshold    float32 `env:"BOOKWORMWIKI_HYPOTHESIS_AUTO_REJECT_THRESHOLD" envDefault:"0.9"`
}

func NewFromFile(filename string) (*Config, error) {
	// Values copied from STASH_* step aside while the .env file loads, so a
	// BOOKWORMWIKI_* entry in the file beats a STASH_* process variable.
	releaseLegacyCopies()
	if _, err := os.Stat(filename); err == nil {
		if err := godotenv.Load(filename); err != nil {
			ApplyLegacyEnvironment()
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		ApplyLegacyEnvironment()
		return nil, err
	}
	// A .env file written for Stash still works: its STASH_* entries are
	// copied to the BOOKWORMWIKI_* names that are not set.
	ApplyLegacyEnvironment()

	cfg := &Config{}
	opts := env.Options{
		RequiredIfNoDef: true,
	}
	if err := env.ParseWithOptions(cfg, opts); err != nil {
		return nil, err
	}
	cfg.applyAuthAliases()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) applyAuthAliases() {
	if c.AuthIssuer == "" {
		c.AuthIssuer = c.AuthOAuthIssuer
	}
	if c.AuthClientID == "" {
		c.AuthClientID = c.AuthOAuthClientID
	}
	if c.AuthClientSecret == "" {
		c.AuthClientSecret = c.AuthOAuthClientSecret
	}
	if c.AuthRedirectURL == "" {
		c.AuthRedirectURL = c.AuthOAuthRedirectURL
	}
	if c.AuthAPISecret == "" {
		c.AuthAPISecret = c.AuthOAuthAPISecret
	}
	if c.AuthStdioToken == "" {
		c.AuthStdioToken = c.AuthOAuthStdioToken
	}
	if raw := strings.TrimSpace(c.AuthOAuthCookieSecureRaw); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			c.AuthCookieSecure = parsed
		}
	}
	if c.AuthOAuthTokenTTL > 0 {
		c.AuthTokenTTL = c.AuthOAuthTokenTTL
	}
}

// Validate rejects configurations that would otherwise fail much later during
// database startup or the first MCP call.
func (c *Config) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.AuthMode)) {
	case "", "none", "oauth", "oidc", "token", "stdio":
	default:
		return fmt.Errorf("BOOKWORMWIKI_AUTH_MODE must be one of none, oauth, oidc, token, or stdio")
	}
	if raw := strings.TrimSpace(c.AuthOAuthCookieSecureRaw); raw != "" {
		if _, err := strconv.ParseBool(raw); err != nil {
			return fmt.Errorf("BOOKWORMWIKI_AUTH_OAUTH_COOKIE_SECURE must be true or false")
		}
	}
	if (strings.TrimSpace(c.AdminUser) == "") != (strings.TrimSpace(c.AdminPassword) == "") {
		return fmt.Errorf("BOOKWORMWIKI_ADMIN_USER and BOOKWORMWIKI_ADMIN_PASSWORD must be set together")
	}
	if c.AdminPassword != "" && (len(c.AdminPassword) < 8 || len(c.AdminPassword) > 72) {
		return fmt.Errorf("BOOKWORMWIKI_ADMIN_PASSWORD must be between 8 and 72 characters")
	}
	if c.VectorDim < 0 || c.VectorDim > 2000 {
		return fmt.Errorf("BOOKWORMWIKI_VECTOR_DIM must be between 0 and 2000")
	}
	if strings.TrimSpace(c.OpenAIBaseURL) == "" && (strings.TrimSpace(c.EmbeddingModel) != "" || strings.TrimSpace(c.ReasonerModel) != "") {
		return fmt.Errorf("BOOKWORMWIKI_OPENAI_BASE_URL is required when BOOKWORMWIKI_EMBEDDING_MODEL or BOOKWORMWIKI_REASONER_MODEL is set")
	}
	if strings.TrimSpace(c.EmbeddingModel) != "" && c.VectorDim <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_VECTOR_DIM must be greater than zero when BOOKWORMWIKI_EMBEDDING_MODEL is set")
	}
	if strings.TrimSpace(c.SecretsKey) != "" {
		if _, err := secrets.ParseKey(c.SecretsKey); err != nil {
			return fmt.Errorf("BOOKWORMWIKI_SECRETS_KEY must be 64 hexadecimal characters (openssl rand -hex 32)")
		}
	}
	for _, previous := range strings.Split(c.SecretsKeyPrevious, ",") {
		if strings.TrimSpace(previous) == "" {
			continue
		}
		if _, err := secrets.ParseKey(previous); err != nil {
			return fmt.Errorf("BOOKWORMWIKI_SECRETS_KEY_PREVIOUS entries must be 64 hexadecimal characters")
		}
	}
	if c.MaxResultSize <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_MAX_RESULT_SIZE must be greater than zero")
	}
	if c.ContextTTL <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_CONTEXT_TTL must be greater than zero")
	}
	if c.EmbeddingRetryInterval <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_EMBEDDING_RETRY_INTERVAL must be greater than zero")
	}
	if c.EmbeddingRetryMaxInterval < c.EmbeddingRetryInterval {
		return fmt.Errorf("BOOKWORMWIKI_EMBEDDING_RETRY_MAX_INTERVAL must be greater than or equal to BOOKWORMWIKI_EMBEDDING_RETRY_INTERVAL")
	}
	if c.EmbeddingRetryBatchSize <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_EMBEDDING_RETRY_BATCH_SIZE must be greater than zero")
	}
	if c.OpenAIRequestTimeout <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_OPENAI_REQUEST_TIMEOUT must be greater than zero")
	}
	if c.ReasonerContextTokens < 0 {
		return fmt.Errorf("BOOKWORMWIKI_REASONER_CONTEXT_TOKENS must not be negative")
	}
	if c.ReasonerReservedTokens < 0 {
		return fmt.Errorf("BOOKWORMWIKI_REASONER_RESERVED_TOKENS must not be negative")
	}
	if c.ReasonerContextTokens > 0 && c.ReasonerContextTokens <= c.ReasonerReservedTokens {
		return fmt.Errorf("BOOKWORMWIKI_REASONER_CONTEXT_TOKENS must be greater than BOOKWORMWIKI_REASONER_RESERVED_TOKENS")
	}
	if c.EmbeddingContextTokens < 0 {
		return fmt.Errorf("BOOKWORMWIKI_EMBEDDING_CONTEXT_TOKENS must not be negative")
	}
	if c.HTTPAddr == "" {
		return fmt.Errorf("BOOKWORMWIKI_HTTP_ADDR must not be empty")
	}
	if c.MCPMaxResponseBytes < 1024 {
		return fmt.Errorf("BOOKWORMWIKI_MCP_MAX_RESPONSE_BYTES must be at least 1024")
	}
	if c.MCPToolTimeout <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_MCP_TOOL_TIMEOUT must be greater than zero")
	}
	if c.AuthTokenTTL < 0 {
		return fmt.Errorf("BOOKWORMWIKI_AUTH_TOKEN_TTL must not be negative")
	}
	if c.AuthSessionTTL < 0 {
		return fmt.Errorf("BOOKWORMWIKI_AUTH_SESSION_TTL must not be negative")
	}
	if c.ConsolidationBatchSize <= 0 {
		return fmt.Errorf("BOOKWORMWIKI_CONSOLIDATION_BATCH_SIZE must be greater than zero")
	}
	if c.ConsolidationSimilarityThreshold < 0 || c.ConsolidationSimilarityThreshold > 1 {
		return fmt.Errorf("BOOKWORMWIKI_CONSOLIDATION_SIMILARITY_THRESHOLD must be between 0 and 1")
	}
	if c.ConsolidationDedupThreshold < 0 || c.ConsolidationDedupThreshold > 1 {
		return fmt.Errorf("BOOKWORMWIKI_CONSOLIDATION_DEDUP_THRESHOLD must be between 0 and 1")
	}
	if c.DecayFactor < 0 || c.DecayFactor > 1 {
		return fmt.Errorf("BOOKWORMWIKI_DECAY_FACTOR must be between 0 and 1")
	}
	if c.ExpiryThreshold < 0 || c.ExpiryThreshold > 1 {
		return fmt.Errorf("BOOKWORMWIKI_EXPIRY_THRESHOLD must be between 0 and 1")
	}
	if c.HypothesisAutoConfirmThreshold < 0 || c.HypothesisAutoConfirmThreshold > 1 {
		return fmt.Errorf("BOOKWORMWIKI_HYPOTHESIS_AUTO_CONFIRM_THRESHOLD must be between 0 and 1")
	}
	if c.HypothesisAutoRejectThreshold < 0 || c.HypothesisAutoRejectThreshold > 1 {
		return fmt.Errorf("BOOKWORMWIKI_HYPOTHESIS_AUTO_REJECT_THRESHOLD must be between 0 and 1")
	}
	return nil
}
