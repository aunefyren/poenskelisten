# Auth Roadmap: MFA, OpenID Connect, Session Revocation & MCP / OAuth Server

Status: **All four phases implemented, shipping in v2.4.0.** This document is now
a design record. Where the shipped code differs from the plan, see
[Where the implementation differs](#where-the-implementation-differs) and the
open bugs in [`wip.md`](./wip.md).

This document plans four phases that harden and modernize Pønskelisten's
authentication:

1. **MFA (TOTP)** — authenticator-app second factor, with admin enforcement. ✅ *shipped*
2. **OpenID Connect (Relying Party)** — "Log in with Authentik/Keycloak/Google". ✅ *shipped*
3. **Refresh tokens + server-side sessions** — short-lived access tokens and real
   revocation/logout. ✅ *shipped*
4. **OAuth 2.1-native authentication + MCP** — make Pønskelisten's own OAuth 2.1
   authorization server the *core* login mechanism: the first-party web app and
   MCP clients both authenticate through it. ✅ *shipped (4.0–4.4)*

Each phase is independently shippable, but the phases are designed as one system:
earlier phases lay groundwork the later ones reuse rather than rework. Read the
[Shared foundations](#shared-foundations) section first — it explains the
decisions that ripple across Phases 1–3.

> **Phase 4 makes OAuth the core auth mechanism — not an opt-in add-on, and not a
> replacement you can skip.** An earlier framing treated OAuth as an additive layer
> for MCP only, leaving the homegrown session login in place. Per decision, Phase 4
> instead rebuilds authentication *on* OAuth: the first-party web login becomes an
> `authorization_code` + PKCE flow, the JSON API becomes an OAuth resource server,
> and MCP clients ride the same authorization server. This **reworks the Phase 1–3
> login/token handling that already shipped** and is by far the largest phase.

---

## Current state (baseline)

- **Homegrown JWT.** `POST /api/open/tokens/register` takes email + password and
  returns an **HS256** JWT signed with the single symmetric `config.PrivateKey`.
- **Claims** (`auth/auth.go`, `JWTClaim`): `first_name`, `last_name`, `email`,
  `admin`, `verified`, `id` (UUID), plus registered claims. 7-day expiry.
- **Sliding refresh.** `/api/auth/tokens/validate` (`controllers/token.go`)
  re-mints the token when it is >24h old; the token is both access and refresh.
- **No server-side session state.** A token cannot be revoked before it expires.
- **Token storage.** Client-side JS-readable cookie `poenskelisten`
  (`web/js/login.js`), refreshed in `web/js/functions.js`.
- **Verification.** When SMTP is enabled, `middlewares/auth.go` forces email
  verification (mints a code, returns 403) for unverified users.
- **User model** (`models/user.go`): `Password *string` is `not null`; `Email`
  is `unique; not null`; soft-delete via `Enabled *bool`.
- **Config** resolves `files/config.json` → flags → env (via `entrypoint.sh`).
  Adding an option touches four places (see CLAUDE.md).

---

## Shared foundations

These are decisions made **once, in Phase 1**, precisely because Phases 2 and 3
depend on them. Getting them right up front avoids migrations-on-migrations.

### F1. User model additions (all added in Phase 1's migration)

Even though some columns are only *used* later, add the nullable columns together
so we run one clean `AutoMigrate` change to `models.User` rather than three:

| Column | Type | Phase used | Purpose |
|---|---|---|---|
| `MFAEnabled` | `*bool` (default false) | 1 | Is TOTP active for this user |
| `MFASecret` | `*string` (nullable) | 1 | TOTP shared secret (encrypted at rest — see F3) |
| `MFAEnrolledAt` | `*time.Time` | 1 | When enrollment completed |
| `OIDCSubject` | `*string` (nullable, indexed) | 2 | IdP `sub` for account linking |
| `OIDCIssuer` | `*string` (nullable) | 2 | Issuer that owns `OIDCSubject` |
| `AuthSource` | `*string` ("local"/"oidc") | 2 | How the account authenticates |

Recovery codes live in a **separate table** (`MFARecoveryCode`: `UserID`,
`CodeHash`, `UsedAt`) so codes are individually consumable — see Phase 1.

**Password nullability.** `Password` stays `*string`, but Phase 2 introduces
OIDC-only users that have no password. Rather than a schema change later, Phase 1
**relaxes the `not null` constraint on `Password` now** (drop `not null` in the
gorm tag) and adds a `hasPassword` helper. Local users still always have one; the
column simply tolerates NULL so Phase 2 doesn't need another migration.

> AutoMigrate does not drop NOT NULL on all drivers reliably. The Phase 1 task
> includes verifying this across sqlite/postgres/mysql and, if needed, a small
> explicit migration step in `database.Migrate()`.

### F2. Admin-editable server settings

Two new admin-controlled settings are introduced across phases:

- `MFAEnforced` (bool) — Phase 1.
- OIDC toggle/config — Phase 2 (mostly static config, not runtime-toggled).

`MFAEnforced` follows the **existing currency-update pattern**
(`APIUpdateCurrency` writes to `config.json`): a config field, an admin
`POST /api/admin/server/settings` endpoint, and exposure through
`APIGetServerInfo` so the frontend can react. This means `MFAEnforced` is a
**config option** and therefore touches the CLAUDE.md "four places"
(`models/config.go`, `config/config.go`, `main.go parseFlags`, `entrypoint.sh`)
plus the README config table.

### F3. Secret handling

- **TOTP secrets** and **OIDC client secret** are sensitive. TOTP secrets are
  stored encrypted at rest using a key derived from `config.PrivateKey`
  (AES-GCM helper in a new `utilities/crypto.go`). This helper is written in
  Phase 1 and reused wherever we persist secrets.
- Recovery codes are **hashed** (bcrypt, same cost as passwords), never stored
  plaintext.

### F4. The "second-step token" pattern (Phase 1) generalizes to Phase 3

Phase 1 introduces a **short-lived, single-purpose challenge token** (used
between password success and TOTP entry). It is a JWT with a distinct `purpose`
claim (`"mfa_challenge"`) and a ~5-minute expiry. Phase 3 reuses the same
"typed token" idea to distinguish **access** vs **refresh** tokens via a
`purpose`/`typ` claim. So Phase 1 adds a `Purpose string` field to `JWTClaim`
(empty = normal session, for backward compatibility) and the validation path
learns to check it. Designing this now means Phase 3 extends an existing field
rather than reworking claim parsing.

### F5. Token issuance stays centralized

All three phases funnel to `auth.GenerateJWTFromClaims`. OIDC (Phase 2) and the
post-MFA success path (Phase 1) both end by minting a normal session token, so
downstream middleware, claims, and frontend are unchanged. Do **not** fork the
session-token format per auth method.

### F6. Testing

Repo has Go tests in `auth/` and `middlewares/` (`*_test.go`) despite CLAUDE.md's
"no tests" note being out of date for those packages. Each phase adds unit tests
for its pure logic (TOTP verify, challenge-token issue/validate, OIDC claim
mapping, refresh rotation). Frontend changes are not unit-tested (no harness) —
call that out per phase.

---

## Phase 1 — MFA (TOTP)

> **Status: implemented.** The enrollment QR code is rendered **server-side**
> (`pquerna/otp` `Key.Image`) and returned as a base64 PNG data URI, so the
> frontend needs no QR library and no external network access; the manual secret
> key + `otpauth://` link are shown as a fallback. Notes:
> - **Recovery codes are an admin option, default off** (`MFARecoveryCodesEnabled`,
>   the second admin-editable setting alongside `MFAEnforced`). When off, no codes
>   are issued at enrollment and the recovery-code login path is disabled even for
>   users who already hold codes — a locked-out user must have their MFA removed by
>   an admin. When on, 10 codes are issued.
> - Recovery codes are hashed with `bcrypt.DefaultCost` (not the password cost)
>   because they carry ~80 bits of entropy, and login verification branches
>   strictly on code shape (6-digit → TOTP, otherwise recovery) to avoid
>   per-attempt bcrypt loops.

**Goal:** users can enroll an authenticator app; login requires the 6-digit code;
admins can enforce enrollment org-wide and delete a user's MFA.

**Dependencies pulled in:** F1 (full user-model migration), F2 (`MFAEnforced`
config), F3 (crypto helper + recovery-code hashing), F4 (challenge token +
`Purpose` claim).

### Library
`github.com/pquerna/otp` (TOTP + provisioning URI/QR). Vet license & add to
`go.mod`.

### Data model
- `models.User`: add the F1 columns (all of them, one migration).
- New `models.MFARecoveryCode` entity + `AutoMigrate` registration in
  `database.Migrate()`.
- `utilities/crypto.go`: AES-GCM encrypt/decrypt using `config.PrivateKey`.

### Backend flow

**Enrollment (authenticated, `/api/auth`):**
1. `POST /api/auth/users/mfa/enroll` — generate secret, store **encrypted** but
   mark `MFAEnabled=false` (pending). Return otpauth URI + base32 secret for QR.
2. `POST /api/auth/users/mfa/activate` — body has a TOTP code; verify against the
   pending secret; on success set `MFAEnabled=true`, `MFAEnrolledAt=now`, generate
   N recovery codes (return plaintext **once**, store hashes).
3. `POST /api/auth/users/mfa/disable` — requires current password **and** a valid
   TOTP/recovery code; clears MFA columns + recovery codes. (Self-service disable,
   distinct from the admin force-delete below.)

**Login (`controllers/token.go`, `GenerateToken`):**
- After password check, if `MFAEnabled`:
  - Do **not** issue a session token. Instead issue an **MFA challenge token**
    (`Purpose="mfa_challenge"`, ~5 min, carries `UserID`).
  - Response signals `mfa_required: true` + the challenge token.
- New `POST /api/open/tokens/mfa` — accepts challenge token + TOTP/recovery code;
  validates; on success issues the normal session JWT (via F5). Recovery code is
  marked `UsedAt`.

**Enforcement (`MFAEnforced`):**
- When `MFAEnforced` is true and a user without MFA logs in, the login succeeds
  but the session is flagged must-enroll. Implement via `middlewares/auth.go`:
  mirror the existing email-verification gate — if `MFAEnforced && !MFAEnabled`,
  return 403 with a `mfa_enrollment_required` marker so the frontend routes the
  user to enrollment. (Enrollment endpoints themselves must remain reachable —
  they are under `/auth`; ensure the gate allow-lists the MFA enroll/activate
  routes, exactly as the verification flow must not lock out its own endpoints.)

**Admin (`/api/admin`):**
- `DELETE /api/admin/users/:user_id/mfa` — `APIAdminDeleteUserMFA`: clears MFA
  columns + recovery codes for that user (recovery/lockout path). Log the action.
- `POST /api/admin/server/settings` — set `MFAEnforced` (F2 pattern, writes
  config.json). Expose `MFAEnforced` via `APIGetServerInfo`.

### Config (four-places + README) — `MFAEnforced bool`
`models/config.go`, `config/config.go` (default false), `main.go parseFlags`
(`-mfaenforced`), `entrypoint.sh` (env `POENSKELISTEN_MFA_ENFORCED`), README table.

### Frontend
- **Account page** (`web/html/account.*`, `web/js/account.js`): enroll UI (show
  QR from otpauth URI — render client-side), activate, view/download recovery
  codes, self-disable.
- **Login** (`web/js/login.js`): handle `mfa_required` → prompt for code → call
  `/open/tokens/mfa`.
- **Admin panel** (`web/js/admin.js`, admin user list): a **"Delete MFA"** button
  per user calling the admin endpoint; a **"Enforce MFA"** toggle calling
  `/admin/server/settings`.
- **Enforcement UX:** on `mfa_enrollment_required`, redirect to the enroll UI.

### Tests
- `auth`: challenge-token issue/validate, `Purpose` gating.
- `utilities`: crypto round-trip.
- MFA verify + recovery-code consumption (mock time/secret).
- Frontend: manual (state per CLAUDE.md).

### Acceptance
- Enroll → logout → login now requires code. Recovery code works once.
- Admin can wipe a locked-out user's MFA. Enforcement pushes non-enrolled users
  to enrollment without locking them out of the enroll endpoints.

---

## Phase 2 — OpenID Connect (Relying Party)

> **Status: implemented.** Decisions taken: account linking to an existing local
> account happens **only when the IdP asserts `email_verified`** (unverified →
> refused); new-user auto-provisioning is a config toggle **default off**
> (`OIDCAutoCreateUsers`); setup docs target **Authelia**. The OIDC client secret
> is stored in `config.json` in plaintext, consistent with the existing
> `SMTPPassword` handling (the F3 crypto helper is used for DB-stored TOTP
> secrets, not config values). State/nonce use short-lived `SameSite=Lax`
> HttpOnly cookies; the session cookie is set by the callback (not HttpOnly, so
> the existing JS reads it — Phase 3 will move to an HttpOnly refresh cookie).
> `UpdateUser` now tolerates passwordless (OIDC) accounts: it skips the
> current-password check and ignores email/password changes for them, so they can
> still update their profile image. Resolution logic lives in
> `database.ResolveOIDCUser` and is unit-tested; `deriveNames` / error mapping are
> tested in `controllers`.

**Goal:** users can log in via an external IdP; a validated IdP login mints a
normal Pønskelisten session JWT (F5). No password for OIDC-only users.

**Dependencies satisfied by Phase 1:** F1 columns (`OIDCSubject`, `OIDCIssuer`,
`AuthSource`) already exist; `Password` already tolerates NULL; crypto helper
(F3) available for the client secret.

### Library
`github.com/coreos/go-oidc/v3/oidc` + `golang.org/x/oauth2`. Discovery via the
issuer's `.well-known/openid-configuration`.

### Config (four-places + README)
New fields: `OIDCEnabled bool`, `OIDCIssuerURL`, `OIDCClientID`,
`OIDCClientSecret`, `OIDCRedirectURL`, optional `OIDCScopes`,
`OIDCAutoCreateUsers bool`. Client secret stored via F3 helper if persisted.
All four places + README.

### Backend flow
- `GET /api/open/oidc/login` — build auth URL with state+nonce (store state in a
  short-lived signed cookie or the session table introduced in Phase 3; for
  Phase 2 use a signed, short-lived cookie), redirect to IdP.
- `GET /api/open/oidc/callback` — validate state, exchange code, **verify ID
  token** (signature via IdP JWKS, `nonce`, `aud`, `exp`), then:
  - **Account linking:** find user by (`OIDCIssuer`,`OIDCSubject`); else by
    verified email → link; else, if `OIDCAutoCreateUsers`, create user with
    `AuthSource="oidc"`, `Verified=true` (IdP-verified), no password.
  - Mint the normal session JWT (F5) and set the `poenskelisten` cookie.
- Interplay with Phase 1: **OIDC users skip local MFA** by default (the IdP owns
  the second factor). `MFAEnforced` must therefore be evaluated as "enforced for
  **local** accounts"; document and implement this in the enforcement gate.

### Middleware interplay
- `middlewares/auth.go`: OIDC-created users are `Verified=true`, so the SMTP
  verification gate is naturally satisfied. Confirm no code path mints an email
  verification code for an `AuthSource="oidc"` user.

### Frontend
- Login page: conditional **"Log in with {IdP name}"** button, shown when
  `OIDCEnabled` (surface via `APIGetServerInfo`).
- Account page: show auth source; hide password-change for OIDC-only users.

### Tests
- ID-token claim → user mapping (table-driven, mock verified claims).
- Account-linking precedence (subject > email > create).
- Frontend: manual.

### Acceptance
- With a test IdP (e.g. local Keycloak/Authentik), login creates/links a user and
  issues a working session. OIDC user has no password and isn't nagged for email
  verification or local MFA.

---

## Phase 3 — Refresh tokens + server-side sessions (revocation)

> **Status: implemented.** Access tokens are short-lived JWTs (15 min,
> `Purpose="access"`); the legacy empty-purpose token is still accepted so the
> upgrade doesn't force everyone to re-login. Refresh tokens are opaque
> (256-bit), stored only as a SHA-256 hash in `models.Session`, and kept in an
> HttpOnly, `SameSite=Lax`, `/api`-scoped cookie. `/open/tokens/refresh` rotates
> on every use with a **10-second grace window** (multi-tab races tolerated) and
> **revoke-all on reuse after grace** (theft response) — all in
> `database.RotateSession`, unit-tested. Logout (`/open/tokens/logout`),
> sign-out-everywhere (`/auth/tokens/logout-all`), and an admin
> `DELETE /admin/users/:id/sessions` revoke sessions. The old sliding-refresh in
> `ValidateToken` is removed. Frontend: `refreshAccessToken` runs on load if the
> access token is missing/expired and on a **10-min proactive timer**; `logout`
> now revokes server-side; the account page has "Sign out of all devices" and the
> admin user modal has "Sign out everywhere".
>
> **Known tradeoff (by design):** access-token validation stays stateless (no DB
> hit on the hot path), so a revoked session's *access* token remains usable until
> it expires (≤15 min). Revocation is enforced at refresh time. TTLs are constants
> (`auth.AccessTokenValidDuration`, `database.RefreshTokenValidDuration`), not yet
> config options.

**Goal:** short-lived access tokens + long-lived, **revocable** refresh tokens
backed by a DB table. Enables real logout and "sign out everywhere". Replaces the
sliding-refresh hack in `controllers/token.go`.

**Dependencies satisfied earlier:** F4 (`Purpose` claim already exists → reuse for
`access` vs `refresh`); F5 (central issuance); the "typed token" validation path
from Phase 1.

### Data model
- New `models.Session` (a.k.a. refresh token): `ID` (UUID), `UserID`,
  `RefreshTokenHash`, `IssuedAt`, `ExpiresAt`, `RevokedAt`, `LastUsedAt`,
  `UserAgent`, `IP`. `AutoMigrate` in `database.Migrate()`.

### Backend flow
- `GenerateToken` (and the Phase 1 `/tokens/mfa` success path, and Phase 2
  callback) now issue **two** tokens: a short-lived access JWT (~15 min,
  `Purpose="access"`) and a refresh token persisted (hashed) in `Session`.
- `POST /api/open/tokens/refresh` — accepts refresh token, validates against
  `Session` (exists, not revoked, not expired), **rotates** it (new refresh,
  revoke old — reuse detection), issues a new access token.
- Replace `/api/auth/tokens/validate` sliding-refresh logic with the refresh
  endpoint. Keep `/validate` for claim inspection but stop re-minting there.
- `POST /api/auth/tokens/logout` — revoke current session.
- `POST /api/auth/tokens/logout-all` — revoke all sessions for the user.
- Admin: optional "revoke user sessions" button (natural companion to Phase 1's
  "Delete MFA").

### Middleware
- `middlewares/auth.go` validates the **access** token (`Purpose="access"`, short
  expiry). No DB hit on the hot path (stateless access token); revocation is
  enforced at refresh time. (Document this tradeoff: access tokens remain valid
  until they expire even after logout; keep access TTL short.)

### Frontend
- `web/js/functions.js` / `login.js`: store both tokens; on 401, transparently
  call `/tokens/refresh`; on refresh failure, redirect to login. Wire logout
  buttons to the new endpoints.
- Consider migrating the refresh token to an **HttpOnly** cookie (server-set) to
  reduce XSS exposure; access token can stay JS-accessible. Note this changes the
  CORS/cookie setup (`AllowCredentials` already true).

### Tests
- Refresh rotation + reuse detection.
- Revocation (single + all).
- Expiry boundaries.

### Acceptance
- Access token expires quickly; refresh keeps sessions alive; logout revokes;
  "logout everywhere" kills all sessions; reused (rotated-away) refresh token is
  rejected.

---

## Phase 4 — OAuth 2.1-native authentication (first-party + MCP)

**Goal:** Pønskelisten runs its own OAuth 2.1 Authorization Server (AS) as the
**single** auth mechanism, in the same binary. The first-party web app is a
**public PKCE client** of that AS; the JSON API and the MCP server are OAuth
**Resource Servers** behind it. There is no separate homegrown login path —
password / MFA / OIDC become the *user-authentication step inside* `/authorize`.

**Resolved decisions (this phase):**
- **Full OAuth-native.** First-party login = `authorization_code` + PKCE (ROPC /
  direct-password grant is disallowed by OAuth 2.1). API + MCP = resource servers
  validating OAuth access tokens.
- **Signing split by role.** The AS keeps a thin **HS256 login-state (SSO) cookie**
  for the `/authorize` step (fast, in-process); **all OAuth-issued tokens — web app
  *and* MCP — are ES256**, verifiable via the JWKS from 4.0.
- **Consent auto-approved for the built-in first-party client**; third-party / MCP
  clients get the consent screen.

**What this reworks vs. the shipped Phases 1–3:**
- The Phase 3 HS256 *access* token, the `poenskelisten` JS cookie, and the
  `POST /api/open/tokens/register` login are **retired for the web app**. The web
  app obtains **ES256 OAuth tokens** via the code flow instead.
- The Phase 3 `Session` + rotation/reuse-detection/revocation machinery is
  **reused** as the OAuth refresh-token store.
- Password/MFA (Phase 1) and OIDC federation (Phase 2) become the login step
  behind `/authorize`; a thin **HS256 SSO session** records "this browser is logged
  in", and the Phase 1–2 MFA-enforcement / email-verification gates apply there.
- `middlewares/auth.go` moves from validating the HS256 session JWT to validating
  **ES256 OAuth access tokens** (`iss`, `aud` == API resource, `exp`, scope).

### Target architecture

```
Browser web app  (built-in PUBLIC PKCE client)
    │ no token → build PKCE verifier/challenge + state, redirect
    ▼
GET /oauth/authorize ── HS256 SSO cookie? ──no──▶ /login (password / MFA / OIDC)
    │ yes / after login                              sets SSO cookie
    ▼
 (first-party client: auto-consent) → single-use code (~60s)
    ▼
POST /oauth/token  (code + PKCE verifier)
    ▼
 ES256 access token (aud = API)  +  rotating refresh (Session)  [+ id_token]
    ▼
web app → /api      ← Resource Server: validate ES256 aud=API + scope
MCP client → /mcp   ← Resource Server: validate ES256 aud=MCP + mcp:* scope
```

**Standards targeted** (pin exact revisions at implementation start): OAuth 2.1
(draft), PKCE (RFC 7636), Authorization Server Metadata (RFC 8414), Protected
Resource Metadata (RFC 9728), Dynamic Client Registration (RFC 7591), Resource
Indicators (RFC 8707), Bearer usage + `WWW-Authenticate` (RFC 6750), JWT access
tokens (RFC 9068), Token Revocation (RFC 7009), Introspection (RFC 7662,
optional), and the **MCP Authorization spec** (verify the current revision at
modelcontextprotocol.io — it is still moving).

This is large; it is split into sub-phases. **4.0 (done) + 4.1 flip the app onto
OAuth; 4.2–4.3 add DCR + the MCP resource server; 4.4 is polish.**

### Sub-phase 4.0 — Foundations (asymmetric keys, metadata, scopes) — ✅ implemented

> **Status: shipped.** Signing is **ES256**; dual-sign — OAuth tokens use the
> asymmetric key while the web session cookie stays HS256. The key is generated on
> first run and persisted in `config.json` (`config/oauthkeys.go`), parsed/cached
> in `auth/oauth.go`, and published at `/.well-known/jwks.json` via go-jose. The
> issuer, algorithm, and API/MCP resource identifiers are **computed from config at
> runtime** (`config.OAuthIssuer()` / `APIResource()` / `MCPResource()`), not
> persisted. Metadata endpoints (`/.well-known/oauth-authorization-server`,
> `/.well-known/oauth-protected-resource`) and JWKS are registered at the domain
> root; AS metadata + JWKS are always served, the protected-resource one gates on
> `MCPEnabled`. Scope registry lives in the new
> `oauth` package. Config guard fails startup if OAuth is enabled without an
> issuer/external URL. All unit-tested (key gen, JWKS has no private component,
> metadata shape + gating). The advertised `/oauth/authorize` and `/oauth/token`
> endpoints arrive in 4.1.
>
> **Adjustment for OAuth-native (do in 4.1):** 4.0 shipped with the AS gated behind
> `OAuthEnabled` (off by default). Because OAuth is now the *core* mechanism, 4.1
> makes the signing key + JWKS + AS metadata **always available** (`OAuthEnabled`
> forced on / deprecated). `MCPEnabled` stays the toggle for the MCP resource
> server + its protected-resource metadata only.

- **G1 — Asymmetric signing + JWKS.** OAuth access/ID tokens must be verifiable by
  parties that don't hold our secret, so **HS256 won't do**. Add an asymmetric
  signer (**ES256** recommended; RS256 acceptable): generate + persist a keypair,
  give it a `kid`, and serve `GET /.well-known/jwks.json`; support ≥2 keys for
  rotation. Keep the Phase-3 first-party session cookie on HS256 (validated
  in-process) and issue **OAuth** tokens with the asymmetric key — two token
  families, distinguished by `iss`/`aud`. Touchpoints: `auth/auth.go` (new signing
  path), `config` (key storage, 0600 file or config.json like `PrivateKey`), new
  `controllers/jwks.go`.
- **G2 — Issuer identity.** `issuer` = `PoenskelistenExternalURL`; all metadata,
  `iss` claims, and discovery derive from it. Add a startup guard requiring it to
  be set (and HTTPS in production) when OAuth is enabled.
- **G3 — Scope registry.** Define grantable scopes and their human descriptions
  (shown on consent). Example: `openid`, `profile`, `email`, plus MCP scopes like
  `mcp:wishlists.read`, `mcp:wishlists.write`, `mcp:groups.read`,
  `mcp:wishes.claim`. Scopes are the unit of consent *and* RS enforcement.
- **G4 — Metadata endpoints** (static JSON derived from config; new
  `controllers/oauth_metadata.go`):
  - `GET /.well-known/oauth-authorization-server` (RFC 8414): issuer, `authorization_endpoint`,
    `token_endpoint`, `registration_endpoint`, `jwks_uri`, `scopes_supported`,
    `response_types_supported` (`code`), `grant_types_supported`
    (`authorization_code`, `refresh_token`), `code_challenge_methods_supported`
    (`S256`), `token_endpoint_auth_methods_supported`.
  - `GET /.well-known/oauth-protected-resource` (RFC 9728): `resource` (the MCP
    server id), `authorization_servers`, `scopes_supported`,
    `bearer_methods_supported` (`header`).
  - **Routing note:** `.well-known/*` lives at the domain root, *outside* `/api` —
    register it in main.go's static/route table, not the `/api` groups.
- **G5 — Config surface** (four-places + README each): `OAuthEnabled`,
  `OAuthIssuer` (default = external URL), signing key material/path, `MCPEnabled`,
  `MCPResourceIdentifier`, DCR-policy toggle (see 4.2). Keep key material and
  secrets out of the admin server-info panel.

### Sub-phase 4.1 — Authorization server core + first-party migration — ✅ implemented

> **Status: shipped** (backend + frontend build/vet/test green; core browser login
> confirmed by the maintainer). Details and the resolved gate-flow fixes are in
> [`phase-4.1-oauth-native.md`](./phase-4.1-oauth-native.md). Remaining: browser
> verification of the SMTP-verify and MFA-enforced first-login paths.
> The planned `APIResourceIdentifier` config option was not added: the API
> resource is always `<issuer>/api` (`config.APIResource()`).

This is the milestone that flips the whole app onto OAuth. It has two halves that
must land together: **(a)** the authorization server, and **(b)** rewiring the
web app + API onto it. Ship behind a feature branch and verify the full login
round-trip before merging (this replaces working login code).

**Also here:** apply the 4.0 adjustment — make the AS always-on; `MCPEnabled` stays
the only toggle (it gates 4.3).

#### (a) Authorization server

**Data models** (embed `GormModel`; register in `Migrate()` + test `allModels()`):
- `OAuthClient`: `ClientID`, `ClientSecretHash` (nullable → public/PKCE client),
  `ClientName`, `RedirectURIs` (exact-match list), `GrantTypes`, `ResponseTypes`,
  `Scopes` (allowed), `TokenEndpointAuthMethod` (`none` | `client_secret_basic` |
  `client_secret_post`), `IsPublic`, `IsFirstParty` (auto-consent), `Registered`
  (DCR vs admin), `Enabled`.
- `AuthorizationCode`: `CodeHash` (store hash only), `ClientID`, `UserID`,
  `RedirectURI`, `Scope`, `Resource`, `CodeChallenge`, `CodeChallengeMethod`
  (`S256`), `ExpiresAt` (~60s), `UsedAt` (single-use).
- `OAuthConsent`: `UserID`, `ClientID`, `Scopes`, granted-at — returning users skip
  consent when scopes are unchanged; revocable ("connected apps").
- **OAuth refresh tokens:** *extend* `models.Session` with `ClientID`, `Scope`,
  `Resource`, and a `Kind` discriminator (`web` | `oauth`) so Phase 3's rotation /
  reuse-detection / revocation are shared.
- **ES256 access-token minting:** new `auth.GenerateOAuthAccessToken(sub, aud,
  scope, …)` signing with the 4.0 key (golang-jwt with the ecdsa key + `kid`
  header). `aud` = the target resource identifier (API or MCP).

**Endpoints** (under `/oauth`, outside `/api`):
- `GET /oauth/authorize`: validate `client_id`, **exact-match** `redirect_uri`,
  `response_type=code`, `scope ⊆ client.Scopes`, `state`, `code_challenge` +
  `S256` (**PKCE required**), `resource` (RFC 8707). Require an authenticated
  resource owner via the **HS256 SSO session**; if absent, redirect to
  `/login?next=…` (which runs password / MFA / OIDC and sets the SSO session), then
  resume. **Auto-approve** when `client.IsFirstParty`; otherwise show the consent
  screen unless an `OAuthConsent` already covers the scopes. On allow → single-use
  `AuthorizationCode` (~60s) → redirect `redirect_uri?code=…&state=…`.
- `POST /oauth/token`:
  - `grant_type=authorization_code`: authenticate the client (secret for
    confidential; PKCE verifier for public), verify the code (unused, unexpired,
    bound to client + redirect_uri), verify `code_verifier` vs `code_challenge`
    (S256), then issue an **ES256 access token** (`iss`, `aud`=`resource`, `scope`,
    `sub`, ~15m) + a rotating **refresh token** (via `Session`) + an **ID token**
    if `openid`. Mark code used.
  - `grant_type=refresh_token`: reuse `database.RotateSession`; re-mint the access
    token with the same/narrowed scope + audience.
- **Consent UI:** templated page (client name + scope descriptions from G3,
  Allow/Deny) → records `OAuthConsent`, resumes code issuance. Skipped for
  first-party.
- **Admin client management:** CRUD for `OAuthClient` in the admin panel (static
  confidential clients), alongside DCR (4.2).

#### (b) First-party client + API as resource server

- **Built-in first-party client:** seed a public PKCE client on migrate — fixed
  `client_id` (e.g. `poenskelisten-web`), `redirect_uri` = `<issuer>/oauth/callback`,
  all first-party scopes, `IsFirstParty=true` (auto-consent).
- **HS256 SSO session:** a small login-state cookie set by `/login`, recording the
  authenticated user for `/oauth/authorize`. The Phase 1–2 MFA-enforcement and
  email-verification gates move here. This is the *only* remaining HS256 token.
- **Frontend rework** (`web/js`): replace `POST /tokens/register` + the
  `poenskelisten` access cookie with the code flow — on load with no access token,
  generate PKCE `verifier`/`challenge` + `state`, redirect to `/oauth/authorize`;
  handle `/oauth/callback` (exchange code at `/oauth/token`, store the ES256 access
  token, keep the refresh token in an HttpOnly cookie as in Phase 3). The Phase 3
  proactive-refresh timer + on-load refresh retarget `/oauth/token`
  (`refresh_token`). Logout → `/oauth/revoke` + clear.
- **API resource server:** `middlewares/auth.go` now validates **ES256 OAuth access
  tokens** (local public key, `iss`, `aud` == API resource, `exp`, required scope)
  instead of the HS256 session JWT. The 4.3 MCP bearer middleware is the *same*
  validator with `aud` = MCP resource. (Retire the legacy HS256 access-token path
  once no old tokens remain, or accept both during a transition window.)

### Sub-phase 4.2 — Dynamic Client Registration (RFC 7591) — ✅ implemented

> **Status: shipped.** Decisions: **open + rate-limited** DCR (10 registrations per
> IP/hour via a new `middlewares.RateLimit`), available whenever `mcp_enabled` is on
> (registration and its `registration_endpoint` are off with it). `POST /oauth/register` creates a
> **public PKCE** client (no secret, `token_endpoint_auth_method=none`), enabled
> immediately, non-first-party so it still hits the consent screen. Validates
> redirect_uris (absolute http(s)), grant types, and scopes (defaults to all
> registry scopes; the user consents to the real ones). Admin can list/revoke
> registered clients (`GET`/`DELETE /api/admin/oauth/clients`, "Connected apps"
> panel; the built-in client can't be revoked). Unit-tested (register happy/validation,
> rate limiter, client CRUD).

- `POST /oauth/register`: accept client metadata (redirect_uris, client_name,
  grant_types, token_endpoint_auth_method, scope), validate, create an
  `OAuthClient` (public/PKCE by default), return `client_id` (+ secret if
  confidential). MCP clients rely on this — they can't be pre-provisioned.
- **DCR policy** (decision + config): open + rate-limited (typical for MCP) vs
  admin-approved vs initial-access-token gated. Open DCR is an abuse surface —
  recommend rate-limiting, PKCE-public-only, exact redirect_uri validation, and
  optionally an admin review queue. Advertise `registration_endpoint` in AS
  metadata only when enabled.

### Sub-phase 4.3 — MCP Resource Server — ✅ implemented

> **Status: shipped.** Decisions: **official Go MCP SDK**
> (`github.com/modelcontextprotocol/go-sdk`) and a **read-only** tool set. New
> `mcpserver` package mounts the SDK's Streamable HTTP handler (stateless, JSON
> responses) at `/mcp`, wrapped in the SDK's `auth.RequireBearerToken` — which
> validates our ES256 tokens (verifier checks `aud` == MCP resource) and emits the
> RFC 9728 `WWW-Authenticate` challenge pointing at
> `/.well-known/oauth-protected-resource`. Self-gates on `MCPEnabled` (404 when
> off). Tools: `list_wishlists`, `list_wishes(wishlist_id)`, `list_groups`, each
> enforcing its `mcp:*` scope from the token and acting as the token's `sub` user
> (owned wishlists only for now). Unit-tested (verifier audience/validity, scope
> helper). Browser/live check with a real MCP client still pending.

- **Bearer middleware:** the **same OAuth access-token validator built in 4.1(b)**,
  parameterised with `aud` == `MCPResourceIdentifier` (rejects token passthrough /
  confused deputy) and the `mcp:*` **scope** required by each operation. The web
  API and the MCP endpoint differ only by expected audience + scopes.
- **401 challenge:** on missing/invalid token respond `401` with
  `WWW-Authenticate: Bearer resource_metadata="<issuer>/.well-known/oauth-protected-resource"`
  (+ `error`, `scope`) — this is how the MCP client discovers the AS and bootstraps
  the flow.
- **MCP endpoint** (`POST /mcp`, Streamable HTTP / JSON-RPC per the MCP spec):
  implement `initialize`, `tools/list`, `tools/call` (optionally `resources/*`).
  Map tools to product capabilities gated by scope — e.g. `list_wishlists`
  (`mcp:wishlists.read`), `add_wish` (`mcp:wishlists.write`), `claim_wish`
  (`mcp:wishes.claim`) — each calling existing `database`/`controllers` logic as
  the token's `sub` user. The tool surface is a product decision that can grow
  incrementally; scope the initial set (decision below).
- **Transport:** decide hand-rolled JSON-RPC + Streamable HTTP vs a Go MCP library.

### Sub-phase 4.4 — Token lifecycle & management — ✅ implemented

> **Status: shipped.** `/oauth/revoke` (RFC 7009) already landed in 4.1.
> **Introspection (RFC 7662) is intentionally omitted** — access tokens are
> self-contained ES256 JWTs validated locally, so nothing consumes opaque tokens.
> Delivered the user-facing **"Connected apps"** on the account page:
> `GET /api/auth/connected-apps` lists the third-party apps a user has authorized
> (from their `OAuthConsent` grants, with human-readable scope descriptions; the
> auto-consenting first-party client never appears), and
> `DELETE /api/auth/connected-apps/:client_id` disconnects one app — removing the
> consent **and** revoking that client's refresh sessions for the user. Distinct
> from the admin "Connected apps" (all registered clients) from 4.2. Unit-tested
> (per-user consent listing, per-client session revoke).

### Sub-phase 4.4 (original plan) — Token lifecycle & management (optional but recommended)

- `POST /oauth/revoke` (RFC 7009): revoke a refresh token / grant (reuses
  `RevokeSessionByRefreshHash`).
- `POST /oauth/introspect` (RFC 7662): optional — JWT access tokens are
  self-contained, so only needed for opaque-token consumers.
- **User "Connected apps":** account-page list of `OAuthConsent` / active OAuth
  sessions with per-app revoke (extends Phase 3's "Sign out of all devices").
  Admin: view/revoke clients and grants.

### Where the implementation differs

These parts of the plan below were built differently. Open gaps are tracked in
[`wip.md`](./wip.md):

- **API audience instead of API scopes.** The API has no per-route scope checks.
  Instead it's reserved for the first-party web app: access tokens carry a
  `client_id` claim (RFC 9068), and `middlewares` refuses API tokens issued to any
  other client. `controllers.resolveResource` applies the same rule when tokens are
  issued: at `/oauth/authorize`, at the consent form (which is re-validated, not
  trusted), and on both token grants. A missing `resource` defaults to the API for
  the first-party client and to MCP for everyone else. Third-party clients can only
  ever get MCP tokens, whose tools enforce their own `mcp:*` scopes.
- **Issuer guard (G2).** There's no hard startup guard. With `external_url` unset,
  the issuer falls back to `http://localhost:<port>`, and startup logs a warning
  instead. Extra login origins can be allowed with `poenskelisten_additional_urls`,
  and logging in from an origin that isn't allowed shows an error page naming the
  address to use.
- **DCR toggle.** There's no separate toggle. Registration (and its
  `registration_endpoint` metadata entry) follows `mcp_enabled`, since
  third-party clients have nothing else to use.
- **Rate limiting.** Only `/oauth/register` is rate-limited. `/oauth/token` and
  `/oauth/authorize` aren't.
- **Key storage.** The signing key lives in `config.json`, which is written `0644`,
  not `0600`.

### Phase 4 security posture
PKCE required (S256 only); exact `redirect_uri` matching; single-use ≤60s codes;
rotating refresh with reuse detection (Phase 3); audience-restricted tokens; no
token passthrough; consent + scope minimization; HTTPS/issuer guard; rate-limit
`/oauth/token`, `/oauth/register`, `/oauth/authorize`; signing keys stored 0600.

### Phase 4 decisions

**Resolved:**
- **Scope:** full OAuth-native — first-party web login is `authorization_code` +
  PKCE; API + MCP are resource servers. (Not opt-in; OAuth becomes core.)
- **Signing:** thin **HS256** SSO login-state cookie for `/authorize`; **all
  OAuth-issued tokens (web + MCP) are ES256** via the 4.0 JWKS.
- **Consent:** auto-approved for the built-in first-party client; shown for
  third-party / MCP clients.
- **Refresh storage:** extend `models.Session` (`ClientID/Scope/Resource/Kind`) —
  reuses Phase 3 rotation/revocation.
- **Deployment:** same binary.

**Resolved during 4.2 / 4.3:**
- **DCR policy:** open + rate-limited (10 registrations per IP per hour).
- **Initial MCP tool set:** read-only `list_wishlists`, `list_wishes` and
  `list_groups`.
- **MCP transport:** the official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk`).

---

## Cross-cutting checklist (every phase)

- [ ] `go build` and `go vet ./...` clean.
- [ ] `gofmt -w` on touched Go files.
- [ ] New config options: **four places** + README table (per CLAUDE.md).
- [ ] `AutoMigrate` additions verified on sqlite/postgres/mysql.
- [ ] User-facing errors stay generic; real error logged via `logger.Log`.
- [ ] Soft-delete / `*bool` sentinel conventions respected.
- [ ] camelCase for new identifiers; migrate touched snake_case where safe.
- [ ] Do not run the binary/browser to verify — ask the user for runtime checks.

## Sequencing rationale

- **Phase 1 first**: highest security value per effort, fully self-contained, and
  it lays every shared foundation (model columns, crypto, typed-token pattern,
  admin-settings pattern) the later phases consume.
- **Phase 2 second**: biggest UX win for self-hosters; slots onto Phase 1's model
  and issuance with no rework.
- **Phase 3 last** (of the auth-hardening set): heaviest lift and the only one
  that changes the token format and frontend token handling; doing it last means
  it absorbs Phases 1–2 rather than being reworked by them.
- **Phase 4 rebuilds on all three**: it reuses the Phase 3 session/rotation infra
  (as the OAuth refresh store) and the Phase 1–2 login (MFA + OIDC) as the
  `/authorize` authentication step — but it also **reworks** the Phase 3 web-app
  token handling onto OAuth, so it's both a build-on and a partial rewrite. Only
  worth starting once 1–3 are stable, which they are. Run as its own
  multi-milestone effort (4.0 done → 4.1 flips the app onto OAuth → 4.2/4.3 add
  DCR + MCP → 4.4 polish).

## Open questions for the user

Phases 1–3 (all resolved during implementation):

1. Recovery codes: how many to issue (default 10)? → 10.
2. Access-token TTL for Phase 3 (default 15 min) and refresh TTL (default 7 days)?
   → 15 min / 7 days.
3. Phase 2: which IdP to test against? → Authelia. `OIDCAutoCreateUsers`? → off.
4. Phase 3: refresh token in an HttpOnly cookie? → yes.

Phase 4: all decisions are **resolved** (full OAuth-native; HS256 SSO cookie + ES256
OAuth tokens; first-party auto-consent; extend `Session`; same binary; open
rate-limited DCR; read-only MCP tools on the official Go SDK). See
[Phase 4 decisions](#phase-4-decisions).
