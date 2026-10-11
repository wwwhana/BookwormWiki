# Local setup with Ollama (no cloud API)

BookwormWiki's landing page promises fully local, fully private memory. This guide closes the gap between marketing and `.env` — **Ollama on your machine, BookwormWiki in Docker, zero outbound API keys.**

Contributed by [Sean Campbell](https://github.com/rudi193-cmd) · tested on Linux with Docker Compose.

---

## What you need

- [Ollama](https://ollama.com) running on the **host** (not inside the BookwormWiki container)
- Docker Compose (BookwormWiki's default install path)
- ~2 GB disk for models + Postgres volume

Pull models **before** `docker compose up`:

```bash
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

| Role | Model | Why |
|------|-------|-----|
| Embeddings | `nomic-embed-text` | 768-dim vectors; matches `BOOKWORMWIKI_VECTOR_DIM=768` |
| Reasoner | `qwen2.5:3b` (or `llama3.2:3b`) | Consolidation synthesis; small models work for dev |

---

## Configure `.env`

Copy `.env.example` → `.env` and use the **Ollama block** below.

BookwormWiki runs in Docker; Ollama runs on the host. From inside the container, reach the host API at:

- **Linux:** `http://host.docker.internal:11434/v1` (Docker 20.10+) or your bridge IP, e.g. `http://172.17.0.1:11434/v1`
- **macOS / Windows Docker Desktop:** `http://host.docker.internal:11434/v1`

```bash
# --- Ollama (fully local) ---
BOOKWORMWIKI_OPENAI_API_KEY=                # leave empty; Ollama does not require a key
BOOKWORMWIKI_OPENAI_BASE_URL=http://host.docker.internal:11434/v1
BOOKWORMWIKI_OPENAI_REQUEST_TIMEOUT=2m
BOOKWORMWIKI_EMBEDDING_MODEL=nomic-embed-text
BOOKWORMWIKI_REASONER_MODEL=qwen2.5:3b
BOOKWORMWIKI_VECTOR_DIM=768                 # must match Ollama's embedding output
# Match these to `ollama show <model>` / the server's configured context.
BOOKWORMWIKI_EMBEDDING_CONTEXT_TOKENS=0
BOOKWORMWIKI_REASONER_CONTEXT_TOKENS=0
BOOKWORMWIKI_REASONER_RESERVED_TOKENS=4096
BOOKWORMWIKI_MCP_TOOL_TIMEOUT=2m
```

> **Changing models later:** Update `BOOKWORMWIKI_EMBEDDING_MODEL` and `BOOKWORMWIKI_VECTOR_DIM`, then restart BookwormWiki. It resizes the pgvector columns, clears old vectors, and re-embeds the saved content in the background. Make a database backup first; the original text is preserved, but vector search is incomplete until reindexing finishes.

---

## Launch

```bash
docker compose up --build
```

Verify:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/sse
docker compose logs bookwormwiki | tail -20
```

Then follow [Getting Started](GETTING_STARTED.md): connect MCP → `init` → `remember` → `recall` → `consolidate`.

---

## Troubleshooting

**`connection refused` to Ollama**

- Confirm Ollama is listening: `curl http://localhost:11434/api/tags`
- On Linux, if `host.docker.internal` fails, add to `docker-compose.yml` under `bookwormwiki`:

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

**Empty recall / embedding errors**

- Model names must match `ollama list` exactly
- `BOOKWORMWIKI_VECTOR_DIM` must match the embedding model (768 for `nomic-embed-text`)

**Consolidation slow or noisy**

- Smaller reasoners (`qwen2.5:3b`, `llama3.2:3b`) are fine for dev; use a larger model only if synthesis quality matters
- If Ollama reports a context error, set `BOOKWORMWIKI_REASONER_CONTEXT_TOKENS` to the model's configured context window. BookwormWiki then splits evidence on paragraph and sentence boundaries. Set `BOOKWORMWIKI_EMBEDDING_CONTEXT_TOKENS` the same way for long passages.

---

## OpenRouter vs Ollama

| | Ollama (this guide) | OpenRouter / cloud |
|---|---------------------|-------------------|
| API key | Not required | Required |
| Privacy | Stays on your LAN | Leaves your network |
| Setup | Host Ollama + Docker bridge | `.env` API key only |
| Best for | Local dev, air-gapped, Willow-style fleets | Quick eval, largest models |

Both paths use the same MCP tools — BookwormWiki is model-agnostic by design.

---

*Questions or fixes:* open an issue on [wwwhana/BookwormWiki](https://github.com/wwwhana/BookwormWiki/issues).
