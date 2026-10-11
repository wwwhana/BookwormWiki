# Migrating from Stash to BookwormWiki

[한국어](#한국어) below.

Stash was renamed to **BookwormWiki**. The repository moved to
[wwwhana/BookwormWiki](https://github.com/wwwhana/BookwormWiki) (GitHub
redirects the old address), the binary is `bookwormwiki`, and every
`STASH_*` setting is now `BOOKWORMWIKI_*`. Your data stays where it is:
namespaces, memory, wiki pages, users, API tokens, and work state are all
read as before. One migration (`00048`) renames the stored owner value of
work resources from `stash` to `bookwormwiki`; it runs on startup like the
others.

## What keeps working without changes

| Stash-era name | What happens now |
| --- | --- |
| `STASH_*` environment variables, in the process or in `.env` | Used for every `BOOKWORMWIKI_*` name that is not set. A new name always wins over an old one, wherever each comes from. When the binary itself reads an old name, it logs one warning listing them. |
| `STASHCONFIG` | Read as `BOOKWORMWIKICONFIG`. |
| `docker-compose.yml` interpolation | Every value falls back from `${BOOKWORMWIKI_X}` to `${STASH_X}`, so an existing `.env` keeps configuring the container. The container only receives the new names, so no warning is logged; check with `grep '^STASH_' .env`. |
| `stash_api_…` API tokens | Accepted on `/mcp`, STDIO, and the token login form until they expire or are revoked. New tokens start with `bookwormwiki_api_`. |
| `stash_session` browser cookie | Accepted. The next request moves the same session (user, expiry) to `bookwormwiki_session`; logging out clears both. |
| `X-Stash-Admin-Token` header | Accepted next to `X-BookwormWiki-Admin-Token`. |
| `stash://work/{id}/brief`, `stash://work-resource/{id}` | Still served next to the `bookwormwiki://` URIs. |
| Work resources owned by the server (`authority: stash`) | Stored rows become `bookwormwiki`; clients that still send `stash` are accepted. |
| `stash.*` keys in a clone's `.git/config` (`repositoryInstanceId`, `projectNamespace`, `remoteName`, …) | Read when the `bookwormwiki.*` key is missing, and copied to it on the next `workspace facts` run that may write. |
| Stored identities (namespace owner keys, worktree keys, remote fingerprints, action receipts) | Computed exactly as before. |
| Theme and language chosen in the console | Carried over while the console address stays the same. |
| PostgreSQL defaults in `docker-compose.yml` | Unchanged (`stash` database and user), so an existing volume keeps working. |

Browsers keep cookies and console preferences per address. If the address
changes (see step 2), people sign in again and pick their theme and language
again; nothing on the server is lost.

## What you change

1. **Keep the same Docker volume.** Compose names volumes after the project,
   which defaults to the directory name. If you rename the checkout from
   `stash` to `BookwormWiki`, pin the old project name first, or the server
   starts on an empty database:

   ```dotenv
   # .env
   COMPOSE_PROJECT_NAME=stash
   ```

2. **Recreate the container.** The compose service and binary are now
   `bookwormwiki`. If you have a `docker-compose.override.yml` or local edits
   that mention the `stash:` service, rename that key to `bookwormwiki:`
   first. Then remove the old container with the stack:

   ```bash
   docker compose up -d --build --remove-orphans
   docker compose exec -T bookwormwiki /bookwormwiki user list
   ```

   With OrbStack the console address follows the service name, for example
   `https://bookwormwiki.stash.orb.local/` instead of `https://stash.stash.orb.local/`.
   If you use SSO, register the new callback URL at the identity provider and
   update the stored provider on the console's **Users & SSO** page or with:

   ```bash
   docker compose exec -T bookwormwiki /bookwormwiki sso list
   docker compose exec -T bookwormwiki /bookwormwiki sso set <id> --redirect-url https://<new-host>/auth/callback
   ```

   `BOOKWORMWIKI_AUTH_REDIRECT_URL` only matters for a provider that was
   never imported into the database.

3. **Rename the MCP server in each client.** The bundled plugin hooks call the
   MCP server named `bookwormwiki`. Existing tokens keep working, and the
   client-side variable name does not matter to the server, so you can keep
   the one you already exported.

   Codex:

   ```bash
   codex mcp remove stash
   codex mcp add bookwormwiki --url https://<host>/mcp --bearer-token-env-var STASH_MCP_TOKEN
   codex plugin marketplace add wwwhana/BookwormWiki
   codex plugin add bookwormwiki-work-plan@bookwormwiki-tools
   ```

   Claude Code:

   ```bash
   claude mcp remove stash
   claude mcp add --transport http bookwormwiki https://<host>/mcp --header "Authorization: Bearer $STASH_MCP_TOKEN"
   claude plugin marketplace add wwwhana/BookwormWiki
   claude plugin install bookwormwiki-work-plan@bookwormwiki-tools
   claude plugin uninstall stash-work-plan@stash-tools
   ```

   Then refresh anything that names the old server: an `AGENTS.md` or
   `CLAUDE.md` copied from `docs/AGENT.md`, and permission allowlists such
   as `mcp__stash__*` (now `mcp__bookwormwiki__*`).

4. **Rename the variables when convenient.** Nothing breaks until you do:

   ```bash
   sed -i.bak -e 's/^STASH_/BOOKWORMWIKI_/' -e 's/^STASHCONFIG=/BOOKWORMWIKICONFIG=/' .env
   ```

5. **Update monitoring.** Prometheus metrics are now `bookwormwiki_*`
   (`bookwormwiki_http_requests_total`, `bookwormwiki_mcp_tool_calls_total`, …).
   Dashboards and alerts on `stash_*` need the new names.

6. **Point your clone at the new repository.**

   ```bash
   git remote set-url origin https://github.com/wwwhana/BookwormWiki.git
   ```

Also renamed: the MCP skill resources (`skill://bookwormwiki-work`,
`skill://bookwormwiki-wiki`), the Go module path
(`github.com/wwwhana/bookwormwiki`), release assets
(`bookwormwiki-<os>-<arch>`), and the test database variables
(`BOOKWORMWIKI_TEST_DATABASE_URL`, `BOOKWORMWIKI_TEST_POSTGRES_DSN`).

---

## 한국어

Stash의 이름이 **BookwormWiki**로 바뀌었습니다. 저장소는
[wwwhana/BookwormWiki](https://github.com/wwwhana/BookwormWiki)로 옮겨졌고
(옛 주소는 GitHub가 리디렉션), 실행 파일은 `bookwormwiki`, 설정 접두사는
`STASH_*`에서 `BOOKWORMWIKI_*`로 바뀌었습니다. 데이터는 그대로입니다.
네임스페이스, 기억, 위키, 사용자, API 토큰, 작업 상태를 모두 이전처럼 읽습니다.
마이그레이션 하나(`00048`)가 작업 리소스의 소유 값 `stash`를 `bookwormwiki`로
바꾸며, 다른 마이그레이션처럼 시작할 때 자동으로 실행됩니다.

### 그대로 동작하는 것

- `STASH_*` 환경 변수(프로세스나 `.env`)는 같은 이름의 `BOOKWORMWIKI_*`가 없을 때 쓰입니다. 어디서 왔든 새 이름이 옛 이름을 이깁니다. 실행 파일이 직접 옛 이름을 읽으면 어떤 것을 썼는지 경고 한 줄을 남깁니다.
- `STASHCONFIG`는 `BOOKWORMWIKICONFIG`로 읽습니다.
- `docker-compose.yml`은 모든 값이 `${BOOKWORMWIKI_X}` → `${STASH_X}` 순으로 채워져 기존 `.env`가 그대로 컨테이너에 들어갑니다. 컨테이너에는 새 이름만 전달되므로 경고가 남지 않습니다. `grep '^STASH_' .env`로 직접 확인하세요.
- `stash_api_…` API 토큰은 만료·폐기 전까지 `/mcp`, STDIO, 토큰 로그인에서 계속 통합니다. 새 토큰은 `bookwormwiki_api_`로 시작합니다.
- `stash_session` 쿠키도 받으며, 다음 요청에서 같은 세션(사용자, 만료 시각)을 `bookwormwiki_session`으로 옮깁니다. 로그아웃은 둘 다 지웁니다.
- `X-Stash-Admin-Token` 헤더도 `X-BookwormWiki-Admin-Token`과 함께 받습니다.
- `stash://work/{id}/brief`, `stash://work-resource/{id}` URI도 `bookwormwiki://`와 함께 계속 제공합니다.
- 서버가 소유한 작업 리소스(`authority: stash`)는 저장된 행이 `bookwormwiki`로 바뀌고, 아직 `stash`를 보내는 클라이언트도 받아들입니다.
- 클론의 `.git/config`에 있는 `stash.*` 키(`repositoryInstanceId`, `projectNamespace`, `remoteName` 등)는 `bookwormwiki.*` 키가 없을 때 읽고, 쓰기가 허용된 다음 `workspace facts` 실행에서 새 키로 복사합니다.
- 저장된 식별값(네임스페이스 소유 키, 워크트리 키, 원격 지문, 작업 영수증)은 이전과 똑같이 계산됩니다.
- 콘솔에서 고른 테마와 언어는 콘솔 주소가 같은 동안 이어집니다.
- `docker-compose.yml`의 PostgreSQL 기본값(`stash` DB·사용자)은 그대로라 기존 볼륨이 계속 쓰입니다.

브라우저는 쿠키와 콘솔 설정을 주소별로 보관합니다. 주소가 바뀌면(2단계) 다시
로그인하고 테마와 언어를 다시 고르면 됩니다. 서버 쪽에서 잃는 것은 없습니다.

### 직접 바꿀 것

1. **같은 Docker 볼륨 유지.** Compose는 볼륨 이름에 프로젝트 이름(기본값은 디렉터리 이름)을 붙입니다. 체크아웃 디렉터리를 `stash`에서 `BookwormWiki`로 바꾼다면 먼저 `.env`에 `COMPOSE_PROJECT_NAME=stash`를 넣으세요. 안 그러면 빈 DB로 뜹니다.
2. **컨테이너 다시 만들기.** 서비스와 실행 파일 이름이 `bookwormwiki`입니다. `docker-compose.override.yml`이나 로컬 수정에 `stash:` 서비스가 있으면 먼저 `bookwormwiki:`로 바꾸세요. 그다음 `docker compose up -d --build --remove-orphans`로 옛 `stash` 컨테이너를 정리하고, 명령은 `docker compose exec -T bookwormwiki /bookwormwiki ...`로 실행합니다. OrbStack 주소도 서비스 이름을 따라 `https://bookwormwiki.stash.orb.local/`처럼 바뀝니다. SSO를 쓴다면 IdP에 새 콜백 URL을 등록하고, DB에 저장된 제공자는 콘솔 **사용자·SSO** 페이지나 `bookwormwiki sso set <id> --redirect-url https://<새 주소>/auth/callback`으로 고칩니다. `BOOKWORMWIKI_AUTH_REDIRECT_URL`은 아직 DB로 가져오지 않은 제공자에만 적용됩니다.
3. **MCP 클라이언트의 서버 이름 변경.** 플러그인 훅은 `bookwormwiki`라는 MCP 서버를 부릅니다. 기존 토큰은 그대로 쓰고, 클라이언트 쪽 변수 이름은 서버와 무관하니 이미 내보낸 변수를 그대로 써도 됩니다.
   - Codex: `codex mcp remove stash`, `codex mcp add bookwormwiki --url https://<host>/mcp --bearer-token-env-var STASH_MCP_TOKEN`, `codex plugin marketplace add wwwhana/BookwormWiki`, `codex plugin add bookwormwiki-work-plan@bookwormwiki-tools`
   - Claude Code: `claude mcp remove stash`, `claude mcp add --transport http bookwormwiki https://<host>/mcp --header "Authorization: Bearer $STASH_MCP_TOKEN"`, `claude plugin marketplace add wwwhana/BookwormWiki`, `claude plugin install bookwormwiki-work-plan@bookwormwiki-tools`, `claude plugin uninstall stash-work-plan@stash-tools`
   - `docs/AGENT.md`를 복사해 둔 `AGENTS.md`·`CLAUDE.md`, `mcp__stash__*` 같은 권한 허용 목록(이제 `mcp__bookwormwiki__*`)도 새 이름으로 고칩니다.
4. **환경 변수 이름은 편할 때 변경.** 바꾸기 전까지 깨지는 것은 없습니다. `sed -i.bak -e 's/^STASH_/BOOKWORMWIKI_/' -e 's/^STASHCONFIG=/BOOKWORMWIKICONFIG=/' .env`
5. **모니터링 갱신.** Prometheus 지표가 `bookwormwiki_*`로 바뀌었습니다. `stash_*`를 쓰는 대시보드와 알림을 고치세요.
6. **원격 저장소 주소 변경.** `git remote set-url origin https://github.com/wwwhana/BookwormWiki.git`

그 밖에 MCP 스킬 리소스(`skill://bookwormwiki-work`, `skill://bookwormwiki-wiki`), Go 모듈 경로(`github.com/wwwhana/bookwormwiki`), 릴리스 파일 이름(`bookwormwiki-<os>-<arch>`), 테스트 DB 변수(`BOOKWORMWIKI_TEST_DATABASE_URL`, `BOOKWORMWIKI_TEST_POSTGRES_DSN`)도 바뀌었습니다.
