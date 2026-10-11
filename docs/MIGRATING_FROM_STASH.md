# Migrating from Stash to BookwormWiki

[한국어](#한국어) below.

Stash was renamed to **BookwormWiki**. The repository moved to
[wwwhana/BookwormWiki](https://github.com/wwwhana/BookwormWiki) (GitHub
redirects the old address), the binary is `bookwormwiki`, and every
`STASH_*` setting is now `BOOKWORMWIKI_*`. Your data does not move: the
database schema, namespaces, wiki pages, users, and API tokens are the same.

## What keeps working without changes

| Stash-era name | What happens now |
| --- | --- |
| `STASH_*` environment variables, in the process or in `.env` | Read when the matching `BOOKWORMWIKI_*` name is not set. The server logs one warning that lists the old names it used. The new name always wins. |
| `STASHCONFIG` | Read as `BOOKWORMWIKICONFIG`. |
| `docker-compose.yml` interpolation | Every value falls back from `${BOOKWORMWIKI_X}` to `${STASH_X}`, so an existing `.env` keeps configuring the container. |
| `stash_api_…` API tokens | Accepted on `/mcp`, STDIO, and the token login form until they expire or are revoked. New tokens start with `bookwormwiki_api_`. |
| `stash_session` browser cookie | Accepted; the first request moves the browser to `bookwormwiki_session`. Logging out clears both. |
| `X-Stash-Admin-Token` header | Accepted next to `X-BookwormWiki-Admin-Token`. |
| Theme and language chosen in the console | Carried over. |
| Namespace ownership | Unchanged: the owner key of every user's namespaces is computed exactly as before. |
| PostgreSQL defaults in `docker-compose.yml` | Unchanged (`stash` database and user), so an existing volume keeps working. |

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
   `bookwormwiki`. Remove the old `stash` container with the stack:

   ```bash
   docker compose up -d --build --remove-orphans
   docker compose exec -T bookwormwiki /bookwormwiki user list
   ```

   With OrbStack the console address follows the service name, for example
   `https://bookwormwiki.stash.orb.local/` instead of `https://stash.stash.orb.local/`.
   Update `BOOKWORMWIKI_AUTH_REDIRECT_URL` and the redirect URL registered at
   your SSO provider if the address changed.

3. **Rename the MCP server in each client.** The bundled plugin hooks call the
   MCP server named `bookwormwiki`. Your existing token keeps working:

   ```bash
   codex mcp remove stash
   export BOOKWORMWIKI_MCP_TOKEN="$STASH_MCP_TOKEN"
   codex mcp add bookwormwiki --url https://<host>/mcp --bearer-token-env-var BOOKWORMWIKI_MCP_TOKEN
   ```

   Reinstall the plugin under its new name:

   ```bash
   codex plugin marketplace add wwwhana/BookwormWiki
   codex plugin add bookwormwiki-work-plan@bookwormwiki-tools
   ```

   Claude Code uses `claude plugin marketplace add wwwhana/BookwormWiki` the
   same way. Remove the old `stash-work-plan` plugin afterwards.

4. **Rename the variables when convenient.** Nothing breaks until you do,
   and the startup warning goes away once you have:

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
`STASH_*`에서 `BOOKWORMWIKI_*`로 바뀌었습니다. 데이터는 그대로입니다. DB
스키마, 네임스페이스, 위키, 사용자, API 토큰 모두 같습니다.

### 그대로 동작하는 것

- `STASH_*` 환경 변수(프로세스나 `.env`)는 같은 이름의 `BOOKWORMWIKI_*`가 없을 때 그대로 읽습니다. 시작할 때 어떤 옛 이름을 썼는지 경고 한 줄로 알려 줍니다. 둘 다 있으면 새 이름이 이깁니다.
- `STASHCONFIG`는 `BOOKWORMWIKICONFIG`로 읽습니다.
- `docker-compose.yml`은 모든 값이 `${BOOKWORMWIKI_X}` → `${STASH_X}` 순으로 채워져 기존 `.env`가 그대로 컨테이너에 들어갑니다.
- `stash_api_…` API 토큰은 만료·폐기 전까지 `/mcp`, STDIO, 토큰 로그인에서 계속 통합니다. 새 토큰은 `bookwormwiki_api_`로 시작합니다.
- `stash_session` 브라우저 쿠키도 받으며, 첫 요청에서 `bookwormwiki_session`으로 옮겨 줍니다. 로그아웃은 둘 다 지웁니다.
- `X-Stash-Admin-Token` 헤더도 `X-BookwormWiki-Admin-Token`과 함께 받습니다.
- 콘솔에서 고른 테마와 언어가 이어집니다.
- 사용자별 네임스페이스 소유 키는 이전과 똑같이 계산되어 기억과 위키가 그대로 보입니다.
- `docker-compose.yml`의 PostgreSQL 기본값(`stash` DB·사용자)은 그대로라 기존 볼륨이 계속 쓰입니다.

### 직접 바꿀 것

1. **같은 Docker 볼륨 유지.** Compose는 볼륨 이름에 프로젝트 이름(기본값은 디렉터리 이름)을 붙입니다. 체크아웃 디렉터리를 `stash`에서 `BookwormWiki`로 바꾼다면 먼저 `.env`에 `COMPOSE_PROJECT_NAME=stash`를 넣으세요. 안 그러면 빈 DB로 뜹니다.
2. **컨테이너 다시 만들기.** 서비스와 실행 파일 이름이 `bookwormwiki`입니다. `docker compose up -d --build --remove-orphans`로 옛 `stash` 컨테이너를 정리하고, 명령은 `docker compose exec -T bookwormwiki /bookwormwiki ...`로 실행합니다. OrbStack 주소도 서비스 이름을 따라 `https://bookwormwiki.stash.orb.local/`처럼 바뀌므로, SSO를 쓴다면 `BOOKWORMWIKI_AUTH_REDIRECT_URL`과 IdP에 등록한 리디렉션 URL도 맞춰 주세요.
3. **MCP 클라이언트의 서버 이름 변경.** 플러그인 훅은 `bookwormwiki`라는 MCP 서버를 부릅니다. 기존 토큰은 그대로 쓰면 됩니다. `codex mcp remove stash` 후 `codex mcp add bookwormwiki --url https://<host>/mcp --bearer-token-env-var BOOKWORMWIKI_MCP_TOKEN`. 플러그인은 `codex plugin marketplace add wwwhana/BookwormWiki`, `codex plugin add bookwormwiki-work-plan@bookwormwiki-tools`로 다시 설치하고 옛 `stash-work-plan`은 지웁니다(Claude Code도 `claude plugin marketplace add wwwhana/BookwormWiki`로 동일).
4. **환경 변수 이름은 편할 때 변경.** 바꾸기 전까지 깨지는 것은 없고, 바꾸면 시작 경고가 사라집니다. `sed -i.bak -e 's/^STASH_/BOOKWORMWIKI_/' -e 's/^STASHCONFIG=/BOOKWORMWIKICONFIG=/' .env`
5. **모니터링 갱신.** Prometheus 지표가 `bookwormwiki_*`로 바뀌었습니다. `stash_*`를 쓰는 대시보드와 알림을 고치세요.
6. **원격 저장소 주소 변경.** `git remote set-url origin https://github.com/wwwhana/BookwormWiki.git`

그 밖에 MCP 스킬 리소스(`skill://bookwormwiki-work`, `skill://bookwormwiki-wiki`), Go 모듈 경로(`github.com/wwwhana/bookwormwiki`), 릴리스 파일 이름(`bookwormwiki-<os>-<arch>`), 테스트 DB 변수(`BOOKWORMWIKI_TEST_DATABASE_URL`, `BOOKWORMWIKI_TEST_POSTGRES_DSN`)도 바뀌었습니다.
