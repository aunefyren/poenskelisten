# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Pønskelisten — a self-hosted wishlist-sharing web app. Go/Gin backend (module `aunefyren/poenskelisten`, Go 1.26), GORM over SQLite/PostgreSQL/MySQL, and a server-rendered vanilla HTML/JS frontend (no SPA framework, no bundler). See `README.md` for end-user features and configuration.

**Read `docs/development.md` before touching any code.** It has the naming, error-handling, and testing conventions this repo expects, and every change should follow them.

**Read `docs/wip.md` at the start of every session.** It tracks known bugs and intentional coverage gaps found but not yet fixed — check it before assuming behavior is correct or coverage gaps are accidental, and keep it in sync (remove an entry once its bug is fixed) as you work.

Never run `git` commands (commit, push, branch, reset, etc.) in this repo — git is managed by the maintainer.

## Commands

Build / vet / format — CI enforces all three, must be clean:
```
go build ./...
go vet ./...
gofmt -l .              # must print nothing; gofmt -w . to fix
```

Tests:
```
go test ./...                                     # whole module
go test ./controllers/...                         # one package
go test ./controllers/... -run TestGroupJoin -v    # one test, verbose
go test -race -covermode=atomic -coverprofile=coverage.out -timeout 20m ./...   # exactly what CI runs
go tool cover -func=coverage.out | tail -1         # print the total %
```
Coverage note: plain `./...` only credits a package with what *its own* test file(s) exercise — that's the number CI's badge/gate use. Add `-coverpkg=./...` to instead credit a package with everything exercised across *all* packages' tests (e.g. `controllers` tests indirectly running code in `database`); the two numbers can differ a lot in this repo.

Run locally:
```
go run .                                                          # serves on :8080, SQLite, by default
./poenskelisten -port 9000 -dbtype postgres -generateinvite true  # flags mirror the env-var table in README.md
```
First run creates `files/config.json`; flags/env vars override it and are persisted back only when something actually changed.

## Architecture

**Layering is strict and one-directional:** `main.go` (wiring + routes) → `controllers` (HTTP handlers, one file per resource) → `database` (GORM queries, one file per resource, filenames mirror `controllers`) → `models` (GORM structs; every persisted struct embeds `models.GormModel` for its UUID PK + timestamps). Controllers never import `gorm.io/*` or touch `database.Instance` directly.

Supporting packages: `auth` (JWT/OAuth token minting + validation), `middlewares` (bearer-token auth gate, rate limiting), `config` (the `config.ConfigFile` global plus load/save and OAuth signing-key generation), `utilities` (crypto, MFA/TOTP, SMTP, text/password validation, legacy-schema migration), `oauth` (the scope registry shared by the authorization server and the MCP server), `oidcprovider` (caches the OIDC relying-party client), `mcpserver` (the MCP resource server), `logger` (logrus wrapper, `logger.Log`).

**Auth model — three unrelated token types, don't conflate them:**
1. **OAuth 2.1 access tokens** (ES256, audience-scoped) — real API auth. The app is its own OAuth 2.1 authorization server (`/oauth/authorize`, `/oauth/token`, `/oauth/revoke`, `/oauth/consent`, PKCE required) *and* a resource server, both for its own API and for MCP. The built-in web client auto-consents; third-party clients (self-registered via `/oauth/register`, e.g. an MCP-speaking AI client) go through a real consent screen. `middlewares.Auth(admin bool)` validates the bearer token on `/api/auth/*` and `/api/admin/*`. The API has no per-route scope checks, so only the first-party client may hold API-audience tokens: `middlewares` checks the token's `client_id` claim, and `controllers.resolveResource` enforces the same rule wherever tokens are issued. Third-party clients only ever get MCP tokens, and registration is off unless `mcp_enabled` is on.
2. **SSO cookie** (HS256) — "this browser is logged in" state, read only inside `/oauth/authorize`. Every login path (password, MFA, OIDC) funnels through `issueSSOSession` (`controllers/session.go`) to set it.
3. **MFA challenge token** (HS256, short-lived) — issued between "password correct" and "TOTP verified"; not a session.

**Route groups** (`main.go`, `initRouter`): `/api/open/*` (no auth — login, register, password reset, public wishlists), `/api/both/*` (shared by authed and public views), `/api/auth/*` (bearer token), `/api/admin/*` (bearer token + admin claim), plus `/oauth/*`, `/.well-known/*`, `/mcp`, and the templated static frontend.

**Frontend is not an SPA.** `web/html/*.html` are Go `html/template` pages, each with a matching `web/js/*.js` (also templated, so it can inline things like the API base URL). Each JS file swaps content into a `#content` div at runtime via `innerHTML` and talks to `/api` with `XMLHttpRequest` — there's no client-side router or bundler. `main.go`'s `registerTemplatedStaticFilesForDirectory` derives a page's URL from its filename automatically; give it an explicit entry in that function's path table only for parameterized routes (`/groups/:group_id`) or when a URL needs to diverge from the filename (see the `/login/mfa` entry, which re-serves `login.html` so the MFA step is a real navigation rather than a DOM swap — some password managers only offer autofill after a real navigation).

**Database:** driver picked by `config.ConfigFile.DBType` (`sqlite`/`postgres`/`mysql`); SQLite always uses the CGO-free `modernc.org/sqlite` driver, since the release binary builds with `CGO_ENABLED=0`. `database.Migrate()` auto-migrates every model and seeds the first-party OAuth client.

## Testing patterns

Tests are colocated (`foo.go` → `foo_test.go`, same package — white-box, so internal helpers get exercised directly). `controllers` tests call handler functions directly rather than through the router: build a context with `gin.CreateTestContext` + `httptest.NewRequest`/`httptest.NewRecorder`, set `ctx.Params`/headers/body as needed, invoke the handler, assert on the recorder. `controllers/helpers_test.go` has the shared fixtures every controller test builds on — `setupControllersDB`, `createTestUser`/`createTestGroup`/`createTestWishlist`/`createTestWish`/..., and `authHeader(t, userID, admin)` (mints a real signed access token, since handlers read the caller via `middlewares.GetAuthUsername`).

**Gotcha:** `config.ConfigFile` is one package-level global shared by every test in a package's binary. A test that changes a field on it (`MFAEnforced`, `PrivateKey`, ...) must restore the original value via `t.Cleanup`, or the change leaks into unrelated tests that happen to run later in the same `go test` invocation.
