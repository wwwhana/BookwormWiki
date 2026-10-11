# BookwormWiki

[🇺🇸 English](README.md) | [🇰🇷 한국어](README.ko.md)

> **Note:** BookwormWiki was called Stash until it was renamed; it is an enhanced fork of the original [alash3al/stash](https://github.com/alash3al/stash). Upgrading from Stash? See [Migrating from Stash](docs/MIGRATING_FROM_STASH.md).
> This version (`wwwhana/BookwormWiki`) introduces significant improvements including Hybrid Search (Vector + Trigram RRF), deterministic episode clustering, safer soft-delete lifecycle, and advanced MCP auto-hydration hooks.

**Your AI has amnesia. We fixed it.**

Every LLM starts every conversation from zero. BookwormWiki gives your agent persistent memory — it remembers, recalls, consolidates, and learns across sessions. No more explaining yourself from scratch.

Open source. Self-hosted. Works with any MCP-compatible agent.

---

## Quick Start

```bash
git clone https://github.com/wwwhana/BookwormWiki.git
cd BookwormWiki
cp .env.example .env
openssl rand -hex 32   # paste this into BOOKWORMWIKI_AUTH_API_SECRET in .env
# edit the provider API key and model settings in .env
docker compose up
```

That's it. Postgres + pgvector, migrations, MCP/metrics servers, and background consolidation — all in one command.

**Next:** [Getting Started guide](docs/GETTING_STARTED.md) — connect your MCP client, run `init` / `remember` / `recall`, and verify everything works.

**Fully local (no cloud API):** [Ollama setup guide](docs/LOCAL_OLLAMA.md) — host Ollama + Docker Compose, private embeddings and reasoner.

## Model Providers

BookwormWiki calls an OpenAI-compatible endpoint for embeddings and reasoning. You can
configure one provider through the environment, or register several in the
database and decide per feature which provider and model to use.

### Environment (quick start)

```bash
BOOKWORMWIKI_OPENAI_BASE_URL=https://api.openai.com/v1
BOOKWORMWIKI_OPENAI_API_KEY=sk-your-openai-api-key
BOOKWORMWIKI_EMBEDDING_MODEL=text-embedding-3-small
BOOKWORMWIKI_REASONER_MODEL=gpt-4o-mini
BOOKWORMWIKI_VECTOR_DIM=1536
```

For a local server such as Ollama or LM Studio, point the base URL at it and
leave the key empty. Match `BOOKWORMWIKI_VECTOR_DIM` to the embedding model's output
size (for example `384` for `multilingual-e5-small`):

```bash
BOOKWORMWIKI_OPENAI_BASE_URL=http://host.docker.internal:11434/v1
BOOKWORMWIKI_OPENAI_API_KEY=
BOOKWORMWIKI_EMBEDDING_MODEL=multilingual-e5-small
BOOKWORMWIKI_REASONER_MODEL=llama3
BOOKWORMWIKI_VECTOR_DIM=384
```

Every model setting is optional. Without an embedding provider BookwormWiki still
stores memories and finds them with trigram keyword search; vectors are
computed later, once a provider is assigned. Without a reasoning provider,
consolidation and plan validation report that no provider is available.

### Provider registry (per-feature routing)

The console's **Model settings** page (`/ui/llm`), the `/admin/llm/*` API, and
the `bookwormwiki llm` command manage providers stored in PostgreSQL. Each feature
(`embedding`, `consolidation`, `plan_validation`, `wiki`) can be assigned its own
provider and model; a feature without an assignment keeps using the
`BOOKWORMWIKI_OPENAI_*` environment. Changes apply immediately without a restart, and
changing the embedding model or dimension resizes the vector columns and queues
a background reindex exactly as an environment change does at startup.

Stored API keys are sealed with AES-256-GCM under `BOOKWORMWIKI_SECRETS_KEY`
(`openssl rand -hex 32`). Without that key, only key-less providers can be
registered. `BOOKWORMWIKI_SECRETS_KEY_PREVIOUS` keeps older keys readable during a
rotation.

```bash
bookwormwiki llm provider add openai --base-url https://api.openai.com/v1 --api-key-env OPENAI_API_KEY
bookwormwiki llm provider probe openai                # lists the models the endpoint exposes
bookwormwiki llm assign embedding --provider openai --model text-embedding-3-small --dimensions 1536
bookwormwiki llm assign plan_validation --provider openai --model gpt-4o
bookwormwiki llm import-env                           # copy BOOKWORMWIKI_OPENAI_* into the registry
bookwormwiki llm status
```

The admin endpoints and the **Server settings** pages are for administrators:
a user whose `is_admin` flag is set (`BOOKWORMWIKI_ADMIN_USER` or `bookwormwiki user set --admin`),
a subject listed in `BOOKWORMWIKI_ADMIN_SUBJECTS`, or a request carrying
`X-BookwormWiki-Admin-Token` (`BOOKWORMWIKI_ADMIN_TOKEN`). Under `BOOKWORMWIKI_AUTH_MODE=none`,
which only listens on loopback, they are open like the rest of the server.
Set `BOOKWORMWIKI_EMBEDDING_CACHE=false` to stop caching computed vectors in PostgreSQL.

## Console login

People sign in to the console with a username and password. On a server
with no account yet, the console opens on a **Create the first
administrator** form: the account made there is the administrator and is
signed in at once. The environment can do the same at startup, which is the
better choice for an unattended deployment because the first-run form is
open to whoever reaches the console first:

```dotenv
BOOKWORMWIKI_ADMIN_USER=admin
BOOKWORMWIKI_ADMIN_PASSWORD=<at least 8 characters>
```

An existing user keeps the password it has (so a change made in the console
survives restarts) but is promoted and re-enabled, which makes these two
variables the way back in. More accounts come from the **Users & SSO** page
of the console (add a user with or without a password, set a password, grant
or remove administrator access, disable, delete, and list or revoke the
person's API tokens by creation date) or from the CLI:

```bash
bookwormwiki user add alice --display-name "Alice" --password-stdin   # or --password-env / --password
bookwormwiki user set alice --admin
bookwormwiki user passwd alice --password-stdin
bookwormwiki user set alice --disable      # takes effect on the next request
bookwormwiki user list                     # users with their identities
```

Users and the ways they sign in are separate tables: a user row is the
person (username, display name, admin flag), and `user_identities` holds its
password hash and any SSO subject. An SSO login whose subject is new is
provisioned as a user named after that subject, so the namespaces and tokens
it already had stay its own, and an administrator can later add a password to
it or disable it. The username is the session subject everywhere: namespaces,
API tokens, and wiki authorship are keyed by it.

The login card shows the password form straight away, a button per
registered SSO provider, and switches to an API-token form on the same card;
nothing sends you to another page first. Signed-in users change their own
password from the **Account** panel.

### SSO providers

SSO providers are OIDC issuers stored in the database (`sso_providers`), with
the client secret sealed by `BOOKWORMWIKI_SECRETS_KEY`. The environment registers
the first one: when `BOOKWORMWIKI_AUTH_ISSUER`, `BOOKWORMWIKI_AUTH_CLIENT_ID`,
`BOOKWORMWIKI_AUTH_CLIENT_SECRET`, and `BOOKWORMWIKI_AUTH_REDIRECT_URL` are set, the server
imports them into the table at startup (once; later edits happen in the
table). From then on administrators manage providers on the **Users & SSO**
page of the console or with the CLI:

```bash
bookwormwiki sso list
bookwormwiki sso add authentik --name "Company SSO" --issuer https://auth.example.com/application/o/bookwormwiki/ \
  --client-id bookwormwiki --client-secret-env SSO_SECRET --redirect-url https://bookwormwiki.example.com/auth/callback
bookwormwiki sso test 1
bookwormwiki sso set 1 --disable
```

Each provider is discovered on load; one that fails discovery is reported on
the page and skipped on the login screen without affecting the others. A
person who signs in through a provider for the first time becomes a user
named after the issuer's subject, so an administrator can later give that
account a password, make it an administrator, or disable it. Without
`BOOKWORMWIKI_SECRETS_KEY` the environment provider still works from memory, but
nothing can be stored.

See [Getting Started](docs/GETTING_STARTED.md) for a fuller configuration checklist.

## MCP Client Setup

After `docker compose up`, BookwormWiki exposes an MCP server over HTTP/SSE. It supports both Streamable HTTP (via `/mcp`) and standard SSE (via `/sse`).

### 1. Claude Code

Use the `/mcp` HTTP endpoint and configure the client to send the local bearer
token generated below.

### 2. Codex

The bundled Docker Compose profile uses token authentication. Set these two
server variables. Generate the signing secret once with `openssl rand -hex 32`
and keep it stable; changing it requires issuing new tokens.

```dotenv
BOOKWORMWIKI_AUTH_MODE=token
BOOKWORMWIKI_AUTH_API_SECRET=<output of openssl rand -hex 32>
```

When BookwormWiki and Codex run on the same machine, issue the token directly into the
current shell and register the MCP server with the exact name `bookwormwiki`:

```bash
export BOOKWORMWIKI_MCP_TOKEN="$(docker compose exec -T bookwormwiki /bookwormwiki mcp token --subject codex --ttl 720h)"
codex mcp add bookwormwiki --url http://127.0.0.1:8080/mcp --bearer-token-env-var BOOKWORMWIKI_MCP_TOKEN
```

For a remote server, issue the token on the server:

```bash
docker compose exec -T bookwormwiki /bookwormwiki mcp token --subject codex --ttl 720h
```

On a Mac running Codex, enter the issued token in zsh without echoing it. The
`export` serves terminal Codex; `launchctl` serves a subsequently launched Codex
app for the current login session:

```zsh
read -s "BOOKWORMWIKI_MCP_TOKEN?BookwormWiki token: "
echo
export BOOKWORMWIKI_MCP_TOKEN
launchctl setenv BOOKWORMWIKI_MCP_TOKEN "$BOOKWORMWIKI_MCP_TOKEN"
codex mcp add bookwormwiki --url https://bookwormwiki.example.com/mcp --bearer-token-env-var BOOKWORMWIKI_MCP_TOKEN
```

Fully quit and reopen the Codex app after setting the variable. The `launchctl`
value does not survive logout or restart.

The work-plan plugin hooks call the MCP server named `bookwormwiki`, so do not choose a
different name. Do not run `codex mcp login bookwormwiki` in token mode. See the
[complete Codex setup](docs/GETTING_STARTED.md#2-connect-your-mcp-client) for
macOS app environment setup, plugin installation, and verification.

You can also use the `stdio` transport and point it to the bookwormwiki CLI binary:

```json
"bookwormwiki": {
  "command": "bookwormwiki",
  "args": ["mcp", "execute", "--with-consolidation", "--consolidate-namespaces", "/projects"]
}
```

For a remote MCP server, use Streamable HTTP with a BookwormWiki API token. Every
MCP client authenticates the same way, whatever the profile: a token that
lives in the database until it expires or is revoked, so an agent's identity
and history never change underneath it. Issue one from the console's **API
tokens** page or on the server:

```bash
export BOOKWORMWIKI_MCP_TOKEN="$(bookwormwiki mcp token --subject codex --name laptop)"
codex mcp add bookwormwiki --url https://bookwormwiki.example.com/mcp --bearer-token-env-var BOOKWORMWIKI_MCP_TOKEN
```

Authentication profiles:

- `none`: no HTTP authentication. BookwormWiki refuses to bind this mode beyond a
  loopback address unless `BOOKWORMWIKI_AUTH_TRUSTED_NETWORK=true` states that the
  network itself is trusted (a private LAN or a VPN); it then logs a warning
  and keeps only the cross-origin protection.
- `oauth` (or the legacy alias `oidc`): adds SSO login to the console through
  an OIDC provider. MCP clients still use BookwormWiki API tokens.
- `token`: username/password and API-token login only; no OIDC provider is
  contacted.
- `stdio`: the local process is trusted, or it can validate a BookwormWiki API token
  given as `BOOKWORMWIKI_AUTH_STDIO_TOKEN` before using an isolated namespace.

HTTP MCP requests must send `Authorization: Bearer <bookwormwiki_api_token>`. Nothing
else is accepted over MCP: no OAuth access token, no upstream identity token.
The browser session cookie is only for the embedded console.

The console's **API tokens** page can issue and revoke BookwormWiki API tokens after
browser login. Choose no expiration, a preset duration (1, 7, 30, 90, or 365 days),
or a custom number of days. Leaving the duration blank means no expiration.
Revoking a token immediately sets its expiration to the revocation time.
The list shows expiration dates and expired tokens;
tokens stop working when they expire or are revoked. The raw value is shown
only once. On the server host, `bookwormwiki mcp token --subject <agent>` stores a
token the same way, so it appears in that subject's token list and can be
revoked there. The lifetime follows `BOOKWORMWIKI_AUTH_TOKEN_TTL` (30 days by
default); `--ttl 0` issues one that lasts until revoked.

### 3. agy (Antigravity)

Configure via `~/.gemini/config/mcp_config.json`:

```json
{
  "mcpServers": {
    "bookwormwiki": {
      "serverUrl": "http://localhost:8080/sse"
    }
  }
}
```

### 4. General MCP Clients (Cursor, Windsurf, OpenCode, Pi, etc.)

Prefer the Streamable HTTP URL `http://localhost:8080/mcp`. Use the SSE URL `http://localhost:8080/sse` only when the client does not support Streamable HTTP.

**Cursor** — `~/.cursor/mcp.json`

```json
{
  "mcpServers": {
    "bookwormwiki": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

**Windsurf** — `~/.codeium/windsurf/mcp_config.json`

```json
{
  "mcpServers": {
    "bookwormwiki": {
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

## Metrics and Health

`bookwormwiki serve` uses one HTTP port for MCP, the web console, login endpoints, metrics, and status checks (default `127.0.0.1:8080`). Docker also publishes port 8080 only on the host loopback interface. Prometheus metrics are available at `http://localhost:8080/metrics`; when HTTP authentication is enabled, `/metrics` requires the same bearer credential as MCP. `/healthz` and `/readyz` stay public for load-balancer probes. The metrics cover HTTP requests, authentication outcomes, MCP tool calls, outbound provider calls, namespace-scope decisions, consolidation backlog and latest errors, terminal result-memory coverage, and pending embedding retries. Request, authentication, tool, provider, and scope metrics use bounded labels and do not include user IDs or raw namespace names.

The HTTP contract is available as OpenAPI at `http://localhost:8080/openapi.json`; the interactive Swagger UI is at `http://localhost:8080/docs` (also `/swagger`). The UI loads its pinned viewer assets from jsDelivr, while the specification remains available same-origin for offline tooling.

If the embedding endpoint still fails after the SDK's short request retries, BookwormWiki saves the raw memory with a pending index state. It also preserves the raw memory when PostgreSQL is reachable but the vector value cannot be stored. The background worker retries without an attempt limit; exponential backoff stops growing at the configured maximum. Configure it with `BOOKWORMWIKI_EMBEDDING_RETRY_INTERVAL`, `BOOKWORMWIKI_EMBEDDING_RETRY_MAX_INTERVAL`, and `BOOKWORMWIKI_EMBEDDING_RETRY_BATCH_SIZE`.

The web console can expose an **Embedding maintenance** page when `BOOKWORMWIKI_ADMIN_SUBJECTS` (comma-separated OIDC subjects) or `BOOKWORMWIKI_ADMIN_TOKEN` is configured. It shows pending work, the current model and dimension, and the latest provider error. **Retry pending** wakes scheduled failures without interrupting active work. **Reindex all** clears stored vectors and the disposable cache, then queues every live episode and fact while preserving the original content.

MCP tool results keep a 32 KiB safety cap by default (this is a transport safeguard, not a model context size). Oversized list pages are split into `items`, `has_more`, and `next_offset`; call the same tool with `offset=next_offset` to continue. Oversized non-list results return a small notice before they can exhaust the client context. Change the cap with `BOOKWORMWIKI_MCP_MAX_RESPONSE_BYTES`.

Reasoner and embedding limits are separate from the MCP response cap. Set `BOOKWORMWIKI_REASONER_CONTEXT_TOKENS` to the full context window of the configured reasoning model and `BOOKWORMWIKI_REASONER_RESERVED_TOKENS` to the space kept for instructions and the JSON answer. Set `BOOKWORMWIKI_EMBEDDING_CONTEXT_TOKENS` to the embedding model's input window. BookwormWiki converts these token budgets to a conservative UTF-8 byte budget, prefers paragraph breaks and then sentence endings, and only hard-splits when no natural boundary fits. Long embedding chunks are combined into one vector. MCP does not publish the model tokenizer or the caller's remaining context; when a limit is left at `0`, BookwormWiki learns a provider-reported limit when possible and adapts after a context-length error. For a 44,544-token reasoner, for example, start with `BOOKWORMWIKI_REASONER_CONTEXT_TOKENS=44544` and a reserve of `4096` or more.

Provider requests and MCP tool handlers have a two-minute default deadline. Set `BOOKWORMWIKI_OPENAI_REQUEST_TIMEOUT` and `BOOKWORMWIKI_MCP_TOOL_TIMEOUT` higher for a slower local model; when a deadline is reached, the call returns an error instead of holding the client until its own timeout, and failed embeddings remain queued for retry. Each row is paused after five failed provider attempts until an operator retries it, so a broken provider cannot consume a small daily quota indefinitely. `recall` still searches saved text with PostgreSQL trigram matching while the embedding provider is unavailable, including a fact's `entity`, `property`, and `value` fields.

When `BOOKWORMWIKI_EMBEDDING_MODEL` or `BOOKWORMWIKI_VECTOR_DIM` changes, BookwormWiki detects the change at startup. It resizes the pgvector columns when needed, clears old vectors and the embedding cache, and queues every live episode and fact for indexing with the new model. The original content is preserved; the background worker recomputes vectors and retries provider failures. Use `bookwormwiki reindex --dry-run` to inspect a manual reindex, or `bookwormwiki reindex` when you want to start one immediately without changing the model setting.

At `info`, BookwormWiki logs completed MCP/API calls and OpenAI-compatible provider calls. Successful `queue_prompt_history` calls stay at `debug` because they occur for every submitted prompt. Set `BOOKWORMWIKI_LOG_LEVEL=debug` to include them and ordinary web access records; failed requests are promoted to `warn`. Logs include the method, bounded path, status, duration, component/model, and request ID when available, but never query strings, authorization headers, cookies, or request bodies.

### Auto-Save &amp; Seamless Handoff

The bundled Codex and Claude Code plugin gives one memory reminder at session startup and queues each submitted prompt without waiting for embedding. It reuses the client's authenticated MCP connection and stores exact prompt text, so review the privacy and opt-out notes in the [**Seamless Agent Handoff Guide**](docs/AGENT_HANDOFF.md).

## What It Does

BookwormWiki is a cognitive layer between your AI agent and the world. Episodes become facts. Facts become relationships. Relationships become patterns. Patterns become wisdom.

A 9-stage consolidation pipeline turns raw observations into structured knowledge — facts, relationships, causal links, patterns, contradictions, goal tracking, failure patterns, and hypothesis verification. Each stage only processes new data since the last run.

## Wiki: the readable layer

Memory is raw: episodes are what happened and facts are what consolidation
concluded. The wiki is what the project currently believes, written down as
Markdown pages that cite the memory and work they were built from. People read
it in the console (`/ui/wiki`, the default page), agents read and write it over
MCP, and `bookwormwiki wiki export` turns a namespace into a folder of `.md` files.

- Pages live per namespace with a `slug` such as `ops/deploys` or
  `decisions/postgres-16`, a kind (article, index, entity, decision, log),
  tags, and a full revision history.
- `[[slug]]` links pages; `[@fact:12]`, `[@episode:3]`, and `[@work:W-000123]`
  cite evidence. Citations are resolved on every read, so a page shows when its
  evidence changed, was superseded, or was deleted, and `wiki_lint` marks such
  pages stale.
- Search works without any model through trigram matching; vectors are added
  when an embedding provider exists. `recall` returns pages next to episodes
  and facts by default.
- Who writes is a choice: agents use `wiki_write` (the bundled `bookwormwiki-wiki`
  skill says when), people edit in the console with a live preview, and
  `wiki_compile` lets the server's reasoner draft a cited page from memory
  when a model is assigned to the `wiki` feature.

MCP tools: `wiki_search`, `wiki_read`, `wiki_list`, `wiki_write`, `wiki_delete`,
`wiki_history`, `wiki_log`, `wiki_lint`, `wiki_compile`. CLI: `bookwormwiki wiki
list|read|write|delete|history|lint|log|export|compile`. See
[docs/WIKI.md](docs/WIKI.md).

## Shared Work Map and Optional Connectors

Work cards live separately from memory data and connect goals, tasks, dependencies, resources, and activity events in one graph. A project can select one shared top-level goal, decompose it into A-1, A-2, and deeper outcomes, and show memory and external resources flowing through work into that shared outcome. The Goal Map shows progress, active agents, blockers, next actions, recent results, and the Jira, Confluence, Git, browser, document, API, data, or device references attached to each item.

The web console's Work Graph lists `/projects/<name>` namespaces as project scopes. Selecting a project limits the graph to that project and its descendants; `All projects` shows work under `/projects` together. MCP clients can pass `project: "/projects/myapp"` to `get_work_graph` to retrieve one project explicitly.

For an owner-facing living plan, use the Work Plan API instead of a changing `PLAN.md`. Its stable component cards hold executable child tasks, directed prerequisites, optional connector references, and decisions made before implementation. `get_work_plan` is the shared current plan; `create_plan_component`, `update_plan_component`, `create_plan_task`, `update_plan_task`, task-state tools, `link_plan_components`, and `record_plan_decision` update it. `validate_work_plan` runs an explicit semantic review with the configured Reasoner model and stores the latest result; it does not use the embedding model. When plan content changes, `get_work_plan.validation.stale` marks the saved review as outdated. The ordinary issue board remains for ad-hoc local issues and does not mix in plan-managed cards.

Call `resume_project(namespace, agent_id, capabilities)` only when continuing an existing BookwormWiki item or a user-requested shared Work Plan. It returns that agent's active work and at most three runnable candidates without a local path, Git repository, or MCP Roots. After choosing an item, call `resume_work` once for a bounded brief containing its goal path, parent plan component, owned scopes, next action, pending conditions, relevant memory, linked resource summaries, completed prerequisite results, and blockers. Do not refresh it unless BookwormWiki reports a conflict or stale state. `claim_work` grants one exclusive lease immediately before action.

During an active lease, `spawn_work` creates a child, prerequisite, or related item with its first action and completion conditions. Child and prerequisite items block the parent until they finish, so many agents can decompose A-1 and A-2 without losing the common outcome. Submit one evidence record for every condition proved by the same observation, then pass the successfully proved pending IDs to `finish_work.passed_condition_ids`. The finish call accepts only IDs backed by current-attempt evidence and is rejected while evidence or blockers are missing.

`attach_work_resource` stores a small reference instead of copying an entire source. A Jira issue or Confluence page can remain authoritative for human work while BookwormWiki records AI work, links the two, and shows both in the Goal Map. The current build provides the neutral resource model; connector polling and write-back are optional add-ons and are not required for the work loop.

Git worktree commands remain available as an optional connector for code projects. They are never a prerequisite for Web MCP work.

```bash
bookwormwiki workspace facts --cwd . --agent-id codex --project-namespace /projects/myapp
bookwormwiki worktree sync --repo . --namespace /projects/myapp
bookwormwiki worktree list --namespaces /projects/myapp
bookwormwiki issue create --namespace / "Login error" --type bug --labels auth,login
bookwormwiki issue list --namespaces / --status doing --label auth
bookwormwiki issue comment add W-000001 --body "Reproduction confirmed"
```

An agent rules sample is available at [docs/AGENT.md](docs/AGENT.md). The default sequence is `resume_project` → `resume_work` → `claim_work` → batched evidence → `finish_work`. Do not call BookwormWiki after routine commands or edits; checkpoint only at interruption, lease-risk, or handoff boundaries. Use `spawn_work` when execution reveals another result that must be delivered.

### Work plan skill

The repository includes the same MCP workflow as a Codex and Claude plugin. Configure a BookwormWiki MCP server named `bookwormwiki`, then install the plugin from this repository. The [complete setup](docs/GETTING_STARTED.md#2-connect-your-mcp-client) covers the server variables, token, MCP registration, plugin, and verification in order:

```bash
claude plugin marketplace add wwwhana/BookwormWiki
claude plugin install bookwormwiki-work-plan@bookwormwiki-tools

codex plugin marketplace add wwwhana/BookwormWiki
codex plugin add bookwormwiki-work-plan@bookwormwiki-tools
```

Claude exposes the plugin skill as `/bookwormwiki-work-plan:bookwormwiki-work-plan`; Codex exposes it as `$bookwormwiki-work-plan`.
Both plugins use the same `hooks/hooks.json`. It adds one local memory reminder
at a new session startup, queues submitted prompts through the authenticated
`bookwormwiki` MCP connection, and requires a successful `finish_work` or
`handoff_work` before claimed work can normally stop. Review and trust the
changed hooks in Codex's `/hooks`; verify that they are listed in Claude Code's
`/hooks`.

The Streamable HTTP endpoint also serves the built-in `bookwormwiki-work` instructions through the experimental `io.modelcontextprotocol/skills` extension described by draft SEP-2640. Clients that implement the draft can discover it with `skills/list`; other clients can keep using the Codex or Claude plugin above. The extension is advertised only on `/mcp`, where its custom methods are reachable.

## License

Apache 2.0
