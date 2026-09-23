# Phase 4.1 — Implementation plan: OAuth-native authentication

Status: **implemented, shipping in v2.4.0.** Core browser login has been verified.
The two gate flows are fixed (see [§14](#14-gate-flows--fixed)) but still need a
browser check with SMTP verification and MFA enforcement turned on. This is the
design record for the sub-phase
that flips Pønskelisten's first-party login onto its own OAuth 2.1 authorization
server. It expands the Phase 4.1 summary in [`auth-roadmap.md`](./auth-roadmap.md).

> **Blast radius warning.** 4.1 rewrites the two hottest pieces of auth: the login
> UX and the per-request API token check. It also **forces a one-time re-login** on
> deploy (the token model changes). Build it on a feature branch and verify the
> full browser round-trip before merge. Treat this doc as the thing to review;
> nothing here is committed until approved.

Builds on the shipped 4.0 (ES256 signing key + JWKS + metadata + scope registry).

---

## 1. Target flow (what we're building)

```
web app (built-in PUBLIC PKCE client)
  1. no access token → make verifier+challenge+state (Web Crypto), save in
     sessionStorage, redirect →
  2. GET /oauth/authorize?client_id=poenskelisten-web&response_type=code
        &redirect_uri=<issuer>/oauth/callback&scope=…&state=…
        &code_challenge=…&code_challenge_method=S256&resource=<API>
        │ SSO cookie? ──no──▶ 302 /login?next=<authorize URL>
        │                      login (password / MFA / OIDC) sets SSO cookie,
        │                      302 back to next
        │ yes → first-party ⇒ auto-consent
        ▼
     302 <redirect_uri>?code=…&state=…
  3. /oauth/callback (web page + JS): verify state, POST /oauth/token
        grant_type=authorization_code, code, code_verifier, client_id, redirect_uri
        ▼
     { access_token (ES256, aud=API, ~15m), id_token, expires_in, scope }
     + Set-Cookie: poenskelisten_refresh=… (HttpOnly, rotating)
  4. store access token; redirect to app; call /api with Bearer access token
```

MCP clients (4.3) run the identical flow with `resource=<MCP>` and `mcp:*` scopes,
except they are confidential/other clients (consent shown; refresh token returned
in the body, not a cookie).

---

## 2. Data model

New entities (embed `GormModel`; register in `database.Migrate()` and the test
`allModels()`), plus an extension of the Phase 3 `Session`.

```
OAuthClient
  ClientID                string  (unique, indexed)   // e.g. "poenskelisten-web"
  ClientSecretHash        *string                      // nil ⇒ public/PKCE client
  ClientName              string
  RedirectURIs            []string (JSON column)       // exact-match allow-list
  Scopes                  []string (JSON column)       // grantable to this client
  GrantTypes              []string (JSON column)       // ["authorization_code","refresh_token"]
  TokenEndpointAuthMethod string                        // "none" | "client_secret_basic" | "client_secret_post"
  IsPublic                bool
  IsFirstParty            bool                          // auto-consent, refresh-in-cookie
  Registered              bool                          // true ⇒ created via DCR (4.2)
  Enabled                 *bool

AuthorizationCode
  CodeHash            string  (indexed)  // SHA-256 of the code; never store the code
  ClientID            string
  UserID              uuid.UUID
  RedirectURI         string
  Scope               string             // space-delimited granted scopes
  Resource            string             // RFC 8707 audience the code is bound to
  CodeChallenge       string
  CodeChallengeMethod string             // "S256"
  ExpiresAt           time.Time          // now + ~60s
  UsedAt              *time.Time          // single-use

OAuthConsent
  UserID    uuid.UUID (indexed)
  ClientID  string    (indexed)
  Scopes    []string (JSON column)
  // unique (UserID, ClientID); updated when scope set changes
```

**Extend `models.Session`** (the Phase 3 refresh store) so OAuth refresh tokens and
SSO sessions reuse rotation/reuse-detection/revocation:

```
Session  (added fields)
  Kind      string   // "web_refresh" | "oauth_refresh" | "sso"
  ClientID  string   // empty for "sso"
  Scope     string
  Resource  string
```

`RotateSession` / `RevokeAllUserSessions` already operate on `Session` — they work
unchanged; new code just sets the new fields on create.

---

## 3. SSO login-state (the only remaining HS256 token)

`/oauth/authorize` needs to know "is this browser logged in, as whom, and are they
verified / MFA-satisfied?" That is the **SSO session**, decided to be **HS256**.

- **Representation:** an HS256 JWT in an **HttpOnly** cookie `poenskelisten_sso`,
  `Purpose="sso"`, `sub=userID`, ~12h expiry. Signed with the existing
  `config.PrivateKey` (reuses `auth` signing). Not the ES256 key.
- **Set by:** the login step. `/login` (password / MFA / OIDC callback) issues it
  instead of the current access token.
- **Read by:** `/oauth/authorize` only. It loads the user to re-check
  `Enabled`/`Verified` and the Phase 1 MFA-enforcement gate lives here now.
- **Global revocation (recommended add):** a `User.SessionsInvalidatedAt *time.Time`
  column; `/authorize` rejects an SSO JWT issued before it. "Sign out everywhere" /
  admin revoke sets it (and revokes OAuth refresh `Session`s). This gives stateless
  SSO cookies a revocation story without a DB row per SSO session.
  - *Alternative* (heavier): make the SSO session a `Session` row (`Kind="sso"`,
    opaque token) validated per `/authorize`. Rejected for now — the HS256 decision
    + `SessionsInvalidatedAt` is simpler and matches the "AS session HS256" choice.

---

## 4. `auth` package additions

- `GenerateOAuthAccessToken(sub uuid.UUID, aud, scope string, admin, verified bool) (string, error)`
  — ES256 JWT signed with the 4.0 key, `kid` header, `iss`=OAuthIssuer,
  `aud`=resource, `scope`, `exp`=now+15m. (golang-jwt supports ES256 with the
  `*ecdsa.PrivateKey` from `config.OAuthSigningKey`.)
- `ValidateOAuthAccessToken(token, expectedAudience string) (*OAuthClaims, error)`
  — parse with the ES256 **public** key, require `iss`==issuer, `aud`==expected,
  `exp`, and return scope + sub. Used by the API and MCP resource servers.
- `GenerateIDToken(...)` — ES256 OIDC ID token (only when `openid` requested).
- `GenerateSSOToken(userID, …) / ValidateSSOToken(...)` — HS256, `Purpose="sso"`.
  (Extends the existing `Purpose` machinery; the API validator must reject
  `Purpose="sso"` just as it rejects `mfa_challenge`.)
- `VerifyPKCE(verifier, challenge string) bool` — `base64url(SHA256(verifier)) ==
  challenge` (S256 only).

---

## 5. `database` layer additions

- `GetOAuthClient(clientID) (OAuthClient, bool, error)`; `ClientHasRedirectURI`,
  `ClientAllowsScopes` helpers (exact-match / subset).
- `CreateAuthorizationCode(...)`, `ConsumeAuthorizationCode(codeHash) (AuthorizationCode, error)`
  (atomic: load unused+unexpired, mark `UsedAt`; reject otherwise).
- `GetConsent(userID, clientID)`, `UpsertConsent(...)`, `RevokeConsent(...)`.
- `SeedFirstPartyClient()` — idempotent create of `poenskelisten-web` on migrate.
- Reuse `CreateSession` (with `Kind/ClientID/Scope/Resource`), `RotateSession`,
  `RevokeSessionByRefreshHash`, `RevokeAllUserSessions`.

---

## 6. Endpoints

All OAuth endpoints under `/oauth` (root, outside `/api`), registered in
`initRouter`. Requests to `/oauth/token` use `application/x-www-form-urlencoded`
(OAuth standard).

### `GET /oauth/authorize`
1. Parse + validate: `client_id` (known, enabled), `redirect_uri` (**exact match**
   against client), `response_type=code`, `scope ⊆ client.Scopes` (via `oauth`
   registry), `state`, `code_challenge` + `code_challenge_method=S256`
   (**required**), `resource` (must be a known resource — API or MCP id).
   - Invalid `client_id`/`redirect_uri` → render an error page (must **not**
     redirect to an unvalidated URI). Other errors → redirect to `redirect_uri`
     with `error=…&state=…`.
2. SSO check: validate `poenskelisten_sso`. If missing/expired/invalidated → `302
   /login?next=<url-encoded current authorize URL>`.
3. Load user; enforce `Enabled`, `Verified` (if SMTP), MFA-enforcement (Phase 1
   gate moved here).
4. Consent: if `client.IsFirstParty` → auto-approve. Else if `GetConsent` already
   covers requested scopes → skip. Else render the consent page (POST →
   `/oauth/consent`).
5. Issue: create single-use `AuthorizationCode` (hash stored, ~60s, bound to
   client+redirect_uri+scope+resource+challenge+user) → `302
   <redirect_uri>?code=…&state=…`.

### `POST /oauth/consent`
Records `UpsertConsent(user, client, scopes)` then continues at step 5. CSRF-protect
(the SSO cookie + a form token).

### `POST /oauth/token`
- `grant_type=authorization_code`: authenticate client (public ⇒ none, but PKCE
  required; confidential ⇒ secret). `ConsumeAuthorizationCode`; verify it is
  bound to this `client_id` + `redirect_uri`; `VerifyPKCE(code_verifier, challenge)`.
  Issue:
  - `access_token`: `GenerateOAuthAccessToken(sub, aud=code.Resource, scope)`.
  - `refresh_token`: `CreateSession(Kind="oauth_refresh", ClientID, Scope,
    Resource)`; **delivery**: first-party ⇒ `Set-Cookie poenskelisten_refresh`
    (HttpOnly, rotating — as Phase 3); other clients ⇒ `refresh_token` in the JSON
    body.
  - `id_token` if `openid` in scope.
  - Body: `{access_token, token_type:"Bearer", expires_in, scope, id_token?}`
    (+ `refresh_token` for non-first-party).
- `grant_type=refresh_token`: source the token (first-party ⇒ cookie; else body);
  `RotateSession`; re-mint access token with same/narrowed scope + same audience;
  re-deliver refresh per client type.

### `POST /oauth/revoke` (RFC 7009) — small, include now for logout
Revoke a refresh token / SSO. Wire logout to it.

### Routing note
`/oauth/*` and the `/oauth/callback` page live outside `/api`; register alongside
the 4.0 `.well-known` routes.

---

## 7. Frontend rework (`web/js`, `web/html`)

The big UX change: login stops being an in-page `POST` and becomes the redirect
flow. Same origin, so it's not a jarring third-party bounce.

- **Bootstrap** (`web/html/*.html` inline + `functions.js get_login`): on load, if
  no valid access token and no refresh, **start the authorize redirect** instead of
  showing the login form. `get_login` keeps validating the access token; on
  missing/expired, try `/oauth/token` refresh; on failure, kick off authorize.
- **PKCE helper** (`functions.js`): `crypto.getRandomValues` for the verifier;
  `crypto.subtle.digest('SHA-256', …)` → base64url challenge. Store
  `verifier`+`state` in `sessionStorage`.
- **`/oauth/callback`** (new `web/html/callback.html` + JS): read `code`+`state`,
  compare `state` to `sessionStorage`, `POST /oauth/token`, store the ES256 access
  token in the `poenskelisten` cookie (still JS-readable; short-lived), clear
  sessionStorage, redirect to the intended page.
- **`login.js`**: the login *page* is now what `/authorize` bounces to
  (`/login?next=…`). `send_log_in` / MFA / OIDC still authenticate, but on success
  the server sets the **SSO cookie** and the page redirects to `next` (back into
  `/authorize`), rather than storing an access token. So `send_log_in`'s success
  branch changes from "store token + go home" to "redirect to `next` (default
  `/`)".
- **Refresh + logout**: `refreshAccessToken` retargets `POST /oauth/token`
  (`grant_type=refresh_token`); the Phase 3 proactive 10-min timer stays. `logout`
  → `POST /oauth/revoke` + clear cookie. "Sign out of all devices" →
  set `SessionsInvalidatedAt` + revoke oauth sessions (existing logout-all path,
  repointed).

---

## 8. API as OAuth resource server (`middlewares/auth.go`)

- `AuthFunction` switches from `auth.ValidateTokenGetClaims` (HS256 session) to
  `auth.ValidateOAuthAccessToken(token, APIResourceIdentifier)` — ES256, `aud`==API
  resource, `exp`, and a baseline scope check. `GetAuthUsername` reads `sub`.
- The MFA-enforcement gate that currently lives in `AuthFunction` **moves to
  `/authorize`** (it's a login-time concern now); the API middleware just checks
  token validity + scope.
- New config `APIResourceIdentifier` (default `<issuer>/api`); the first-party
  client requests `resource=<API>` so its tokens carry `aud`=API.
  *As shipped:* no config option (the resource is always `config.APIResource()`),
  and no per-route scope check. Instead, API tokens are only valid when issued to
  the first-party client (their `client_id` claim is checked). See "Where the
  implementation differs" in [`auth-roadmap.md`](./auth-roadmap.md).
- 4.3's MCP middleware is the same validator with `aud`=MCP resource.

---

## 9. Config & seeding

New config (four-places + README): `APIResourceIdentifier`. **Flip 4.0's gate:**
OAuth is now core, so the signing key / JWKS / AS metadata are **always served**
(deprecate `OAuthEnabled`, or force it true); `MCPEnabled` stays (gates 4.3 only).
On migrate, `SeedFirstPartyClient()` creates `poenskelisten-web`
(public, PKCE, `redirect_uri=<issuer>/oauth/callback`, first-party scopes,
`IsFirstParty=true`). Startup guard (from 4.0) already requires an issuer.

---

## 10. Migration / backward-compatibility

- **One-time re-login on deploy** (accepted): the token model changes, so existing
  Phase 3 HS256 access tokens + refresh cookies stop working. On first load users
  hit `/authorize` → no SSO cookie → `/login`. Communicate this in the release.
- *Optional soft landing* (only if wanted): temporarily accept the old Phase 3
  refresh cookie at a shim endpoint to mint an SSO session, avoiding the re-login.
  Adds complexity; recommend skipping.
- Old `POST /api/open/tokens/register`, `/tokens/refresh`, `/tokens/mfa` remain for
  a deprecation window or are removed once the frontend no longer calls them.

---

## 11. Suggested implementation order

1. Models + migration + `Session` extension + `SeedFirstPartyClient` + config
   (`APIResourceIdentifier`, always-on OAuth) + `User.SessionsInvalidatedAt`.
2. `auth`: ES256 access/ID token mint + `ValidateOAuthAccessToken`; SSO
   mint/validate; `VerifyPKCE`. (Unit-test heavily — cheap and central.)
3. `database`: client/code/consent helpers.
4. Controllers: `/oauth/authorize`, `/oauth/consent`, `/oauth/token`,
   `/oauth/revoke`; SSO cookie set in the login paths.
5. `middlewares/auth.go`: API resource-server validation; move the MFA gate to
   `/authorize`.
6. Frontend: PKCE + authorize redirect, `/oauth/callback`, login-page redirect,
   refresh/logout retarget.
7. Flip 4.0 gating to always-on; retire/990-deprecate the old token endpoints.
8. Tests + full manual round-trip.

## 12. Testing

- **Unit:** `VerifyPKCE`; ES256 mint/validate (aud match/mismatch, expiry, scope);
  SSO mint/validate + `SessionsInvalidatedAt`; code single-use + expiry;
  redirect_uri exact match; consent upsert; `RotateSession` with `Kind`.
- **Handler-level** (gin test context, like the 4.0 metadata tests): `/authorize`
  validation + redirect-to-login + code issuance; `/token` happy path, PKCE
  mismatch, reused code, wrong client, wrong redirect_uri.
- **Manual (user):** full browser login round-trip; refresh keeps session; logout
  revokes; MFA prompted inside the flow; "Log in with Authelia" inside the flow;
  a second client (fake MCP) gets the consent screen and a working token.

## 14. Gate flows — fixed

Both pre-token *gate* flows now identify the user from the login (SSO) session, so
they work before the user holds an OAuth access token. Shared helper
`resolveGateUser` accepts either an access token (voluntary use, e.g. MFA setup
from the account page) or the SSO cookie (gated before tokens).

- **Email verification gate** (SMTP enabled): `/oauth/authorize` redirects an
  unverified user to `/verify`; `VerifyUser` + `SendUserVerificationCode` now use
  `resolveGateUser`, and the verify page submits with the SSO cookie (no access
  token). On success → `/` → the OAuth flow completes.
- **MFA enrollment gate** (MFA enforced): `/oauth/authorize` redirects an
  unenrolled local user to a dedicated **`/enroll`** page (not `/account`, which
  looped). Enrollment endpoints (`/api/open/users/mfa/enroll` + `/activate`) moved
  under `/open` with SSO-cookie identification. On activation → `/` → tokens.

Verified end-to-end paths still needing a **browser check with those settings on**:
register → login → verify → app; and MFA-enforced first login → `/enroll` → app.

## 13. Risks / decisions to confirm before coding

1. **One-time re-login on deploy** — acceptable? (Recommended: yes; alternative is
   the soft-landing shim.)
2. **Refresh delivery split** — first-party ⇒ HttpOnly cookie (keeps Phase 3 XSS
   protection); other clients ⇒ body. Confirm the deviation from "refresh always in
   body" is fine (recommended).
3. **SSO revocation** via `User.SessionsInvalidatedAt` vs a `Session`-row SSO.
   (Recommended: the timestamp.)
4. **Old token endpoints** — deprecate-with-window vs remove in the same PR.
