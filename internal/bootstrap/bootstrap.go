package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wwwhana/bookwormwiki/internal/auth"
	"github.com/wwwhana/bookwormwiki/internal/brain"
	"github.com/wwwhana/bookwormwiki/internal/config"
	"github.com/wwwhana/bookwormwiki/internal/db"
	"github.com/wwwhana/bookwormwiki/internal/llm"
	"github.com/wwwhana/bookwormwiki/internal/queries"
	"github.com/wwwhana/bookwormwiki/internal/secrets"
)

// Context holds all initialized services.
type Context struct {
	Config *config.Config
	Auth   *auth.Provider
	Brain  *brain.Brain
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	// LLM resolves each feature to a provider; LLMStore edits that mapping.
	LLM      *llm.Router
	LLMStore *llm.Store
}

// New initializes all services: database, model routing, queries, brain.
func New(ctx context.Context) (*Context, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := buildLogger(cfg)
	if legacy := config.LegacyEnvironmentNames(); len(legacy) > 0 {
		logger.Warn("deprecated Stash environment variables are in use; rename STASH_ to BOOKWORMWIKI_ (and STASHCONFIG to BOOKWORMWIKICONFIG)", "variables", strings.Join(legacy, ","))
	}

	authProvider, err := auth.Init(ctx, auth.Config{
		Mode:          cfg.AuthMode,
		Issuer:        cfg.AuthIssuer,
		ClientID:      cfg.AuthClientID,
		ClientSecret:  cfg.AuthClientSecret,
		RedirectURL:   cfg.AuthRedirectURL,
		APISecret:     cfg.AuthAPISecret,
		CookieSecure:  cfg.AuthCookieSecure,
		APITokenTTL:   cfg.AuthTokenTTL,
		SessionTTL:    cfg.AuthSessionTTL,
		StdioToken:    cfg.AuthStdioToken,
		AdminSubjects: cfg.AdminSubjects,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize authentication: %w", err)
	}

	pool, err := db.OpenPool(ctx, cfg.StoreDSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if authProvider != nil {
		authProvider.SetTokenPool(pool)
	}
	if strings.TrimSpace(cfg.AdminUser) != "" {
		if authProvider == nil {
			logger.Warn("BOOKWORMWIKI_ADMIN_USER is set but BOOKWORMWIKI_AUTH_MODE=none performs no login; the account is not created")
		} else if authProvider.Mode() == "stdio" {
			logger.Warn("BOOKWORMWIKI_ADMIN_USER is ignored in stdio mode")
		} else {
			created, err := authProvider.SeedLocalAdmin(ctx, cfg.AdminUser, cfg.AdminPassword)
			if err != nil {
				pool.Close()
				return nil, fmt.Errorf("seed local administrator: %w", err)
			}
			if created {
				logger.Info("local administrator created from BOOKWORMWIKI_ADMIN_USER", "username", strings.ToLower(strings.TrimSpace(cfg.AdminUser)))
			} else {
				logger.Info("local administrator exists; BOOKWORMWIKI_ADMIN_PASSWORD was not applied", "username", strings.ToLower(strings.TrimSpace(cfg.AdminUser)))
			}
		}
	}

	if authProvider != nil && authProvider.Mode() != "stdio" && strings.TrimSpace(cfg.AdminUser) == "" && authProvider.SetupRequired(ctx) {
		logger.Warn("no user account exists yet; the first visitor of the console creates the administrator, or set BOOKWORMWIKI_ADMIN_USER and BOOKWORMWIKI_ADMIN_PASSWORD")
	}
	keyring, err := secrets.NewKeyring(cfg.SecretsKey, strings.Split(cfg.SecretsKeyPrevious, ",")...)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("load secrets key: %w", err)
	}
	if keyring == nil {
		logger.Warn("BOOKWORMWIKI_SECRETS_KEY is not set; provider API keys and SSO client secrets cannot be stored in the database")
	}
	if authProvider != nil && authProvider.Mode() != "stdio" {
		authProvider.SetSecrets(keyring)
		// The environment registers the first SSO provider; afterwards the
		// table is the source of truth and the console edits it.
		if imported, ok, err := authProvider.ImportEnvironmentSSO(ctx); err != nil {
			logger.Warn("SSO provider from the environment was not imported", "error", err)
		} else if ok {
			logger.Info("SSO provider imported from the environment", "slug", imported.Slug, "issuer", imported.Issuer)
		}
		if err := authProvider.ReloadSSO(ctx); err != nil {
			logger.Warn("load SSO providers", "error", err)
		}
	}

	store := llm.NewStore(pool, keyring)
	router := llm.NewRouter(pool, store, envProvider(cfg), llm.Options{
		Logger:         logger,
		EmbeddingCache: cfg.EmbeddingCache,
		PrepareStorage: func(ctx context.Context, model string, dims int) (db.EmbeddingStorageReport, error) {
			return db.PrepareEmbeddingStorage(ctx, pool, model, dims)
		},
	})
	if err := router.Reload(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("load model routing: %w", err)
	}
	logger.Info("model input limits",
		"reasoner_context_tokens", cfg.ReasonerContextTokens,
		"reasoner_reserved_tokens", cfg.ReasonerReservedTokens,
		"embedding_context_tokens", cfg.EmbeddingContextTokens,
		"embedding_cache", cfg.EmbeddingCache,
		"openai_request_timeout", cfg.OpenAIRequestTimeout,
		"mcp_tool_timeout", cfg.MCPToolTimeout,
	)

	q, err := queries.New()
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("load queries: %w", err)
	}

	window, err := time.ParseDuration(cfg.ConsolidationWindow)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("parse consolidation window: %w", err)
	}

	br, err := brain.New(pool, router.Embedder(), router.Reasoner(), q, brain.Config{
		MaxResultSize:                  cfg.MaxResultSize,
		BatchSize:                      cfg.ConsolidationBatchSize,
		SimilarityThreshold:            cfg.ConsolidationSimilarityThreshold,
		DedupThreshold:                 cfg.ConsolidationDedupThreshold,
		Window:                         window,
		DecayFactor:                    cfg.DecayFactor,
		ExpiryThreshold:                cfg.ExpiryThreshold,
		HypothesisAutoConfirmThreshold: cfg.HypothesisAutoConfirmThreshold,
		HypothesisAutoRejectThreshold:  cfg.HypothesisAutoRejectThreshold,
		EmbeddingRetryInterval:         cfg.EmbeddingRetryInterval,
		EmbeddingRetryMaxInterval:      cfg.EmbeddingRetryMaxInterval,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("build brain: %w", err)
	}

	return &Context{
		Config:   cfg,
		Auth:     authProvider,
		Brain:    br,
		Pool:     pool,
		Logger:   logger,
		LLM:      router,
		LLMStore: store,
	}, nil
}

// envProvider turns the BOOKWORMWIKI_OPENAI_* variables into the fallback provider.
// It is nil when no base URL is configured, which makes every feature depend
// on database assignments alone.
func envProvider(cfg *config.Config) *llm.EnvProvider {
	if strings.TrimSpace(cfg.OpenAIBaseURL) == "" {
		return nil
	}
	return &llm.EnvProvider{
		BaseURL:                cfg.OpenAIBaseURL,
		APIKey:                 cfg.OpenAIAPIKey,
		RequestTimeout:         cfg.OpenAIRequestTimeout,
		EmbeddingModel:         cfg.EmbeddingModel,
		VectorDim:              cfg.VectorDim,
		EmbeddingContextTokens: cfg.EmbeddingContextTokens,
		ReasonerModel:          cfg.ReasonerModel,
		ReasonerContextTokens:  cfg.ReasonerContextTokens,
		ReasonerReservedTokens: cfg.ReasonerReservedTokens,
	}
}

// Close releases all resources.
func (c *Context) Close() error {
	var errs []string
	if c.Brain != nil {
		c.Brain.Close()
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func loadConfig() (*config.Config, error) {
	config.ApplyLegacyEnvironment()
	filename := os.Getenv("BOOKWORMWIKICONFIG")
	if filename == "" {
		filename = ".env"
	}
	return config.NewFromFile(filename)
}

func buildLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{}

	switch cfg.LogLevel {
	case "debug":
		opts.Level = slog.LevelDebug
	case "info":
		opts.Level = slog.LevelInfo
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		opts.Level = slog.LevelInfo
	}

	// Logs go to stderr so a command's stdout stays machine-readable: for
	// example `bookwormwiki mcp token` prints only the token, and the JSON commands
	// print only JSON. Container log collectors read both streams.
	if cfg.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}
