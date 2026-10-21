---
domain: Designs
status: Active
entry_points:
  - api/openapi.yaml
  - cmd/api.go
  - cmd/web_runtime.go
  - accountgame/accountgame.go
  - game/serialization.go
  - db/accounts.go
dependencies:
  - .aidoc/designs/web-api.md
  - .aidoc/designs/database-catalog.md
  - .aidoc/designs/e2e-api-scenarios.md
---

# User Accounts and Guest Play

Sudoku adopts optional Google accounts while preserving immediate guest play. The backend keeps guest state stateless and sealed, owns authenticated identity and game authorization, and exposes one explicit transition from the browser's single guest game to an account game.

## Related Docs

| Document | Relationship |
|----------|--------------|
| `.aidoc/designs/web-api.md` | Existing client-neutral HTTP and revision boundary |
| `.aidoc/designs/database-catalog.md` | Shared puzzle catalog and presentation-specific play runs |
| `.aidoc/designs/e2e-api-scenarios.md` | Built-binary HTTP acceptance catalog |
| [Sudoku UI account experience](https://github.com/gnailuy/sudoku-ui/blob/master/.aidoc/designs/user-accounts.md) | Browser storage, sign-in, claim, and account surfaces |

## Why Optional Accounts Exist

Sudoku must remain playable before identity setup. A guest can keep one active game through refresh and browser restart on the same device; signing in adds cross-device continuity and a My games surface without silently converting browser data.

Guest play must create no durable player, recovery, session, or analytics row on the server. This constraint avoids pseudo-accounts and server-side tracking before consent while retaining the authoritative Go game engine for every action.

## Ownership States

A guest game is one opaque sealed document stored by the browser. The sealed plaintext contains a random document identifier, exact puzzle presentation, complete current engine state and history, revision, actual difficulty, issued time, and expiry; authenticated encryption prevents browser forgery or authoritative edits.

An account game belongs to exactly one `users` row, references the shared base-puzzle catalog and a presentation-specific play run, and stores durable engine state. Every account-game read, mutation, list, and deletion resolves ownership from the authenticated application session rather than accepting a user identifier from the client.

A browser may retain at most one guest game. The backend therefore exposes no guest collection, guest list, guest recovery row, or bulk-import operation. Starting another guest game replaces the browser's prior guest document only after the new document is returned successfully.

## Identity and Session Model

`users` represents the application account. `external_identities` binds the normalized Google issuer and subject pair uniquely to one user; profile email and display fields are mutable attributes and never identity keys.

`application_sessions` stores a random verifier digest, user ID, creation and last-seen times, idle and absolute expiry, rotation lineage, and revocation time. Idle expiry defaults to 30 days and revocation takes effect immediately. The browser receives only an opaque `Secure`, `HttpOnly`, `SameSite=Lax` cookie with the narrowest application path; Google tokens and application bearer tokens never enter browser storage.

Google authentication uses authorization code with PKCE, state, and nonce. The backend creates short-lived, single-use login transactions, validates the exact issuer, audience, redirect URI, state, nonce, PKCE verifier, and token times, rotates any existing application session after callback, then returns the browser to an allowlisted relative destination.

Logout revokes the current application session. Account settings can revoke all application sessions and delete the account; account deletion revokes sessions and removes identities and owned games while preserving the shared puzzle catalog.

## Guest Document Boundary

Guest create and action endpoints accept no account game ID and never infer ownership mode from a cookie. Each action submits the latest sealed document, expected revision, and one existing engine action; the backend authenticates and opens the document, validates schema and expiry, applies the action through `game.Game`, increments revision once, and returns a replacement sealed document plus the authoritative snapshot.

Malformed, tampered, expired, or unsupported guest documents fail closed with stable recoverable errors. An older valid document may still be accepted because guest play has no competitive reward or server record; the claim boundary supplies replay protection where durable ownership begins.

Plain `sudoku api` creates a random 32-byte sealing key on first start, installs it atomically as a private file in the managed application-data directory, and reuses it across restarts. Unix uses owner-only mode bits. Windows uses a protected DACL limited to the current user, Local System, and Administrators and rejects reparse points or any additional allowed principal. An explicit web configuration must satisfy the same platform-native boundary before it can replace the local key with named active and bounded decrypt-only rotation keys. The project ships no compiled shared key and never logs sealed documents, OAuth material, session cookies, puzzle history, or profile data.

## Explicit Claim Transaction

After sign-in, the browser may submit its one current guest document to the claim endpoint. One database transaction authenticates and validates the latest document, creates the owned account game and linked play-run state, records a unique claim fingerprint derived from the authenticated guest document, and returns the durable account game.

The claim fingerprint makes retries idempotent for the same user and prevents one guest document from creating duplicate durable games. A claim by another user fails without revealing the first owner. The browser deletes its guest record only after a successful claim response; cancellation or failure leaves the local copy intact.

Already authenticated players create account-owned games from the beginning. The shared catalog row is prepared first, then one concrete SQLite transaction inserts the presentation-specific play run and its account game together; failure of either linked insert leaves neither row. Account endpoints never accept sealed guest state except at claim, and guest endpoints never switch to account mode merely because a application-session cookie is present.

## API and Persistence Shape

Current SQLite schema version 6 stores provider-keyed identities, digest-only application sessions, owner-scoped account games, and independently bound serving-cohort evidence. The explicit `sudoku db rebuild-web` cutover replaces obsolete browser-private identity and game state while preserving catalog and cohort evidence. Account games link shared base puzzles to presentation-specific play runs, retain complete engine state with optimistic revisions, authoritative status, and monotonic player elapsed time, and disappear with their owning user while catalog rows and play-run history remain.

The OpenAPI contract names guest and account resources separately. Guest operations create and mutate sealed documents; authentication operations start login, complete callback, report the current account, log out, and revoke sessions; account-game operations list status/time summaries, create, read, mutate, synchronize elapsed presentation time, delete one or all owned games, and claim one guest document. `oidcauth.Manager` owns short-lived single-use state, nonce, PKCE, return-path, issuer, audience, and token-time validation, while `oidcauth.GoogleProvider` owns discovery, code exchange, signature verification, and profile-claim extraction. Plain `sudoku api` enables only the guest capability and performs no Google discovery. An owner-only `--web-config` may provide only the nested guest section or both the guest and account sections. The complete account section supplies an independent CSRF key, Google client configuration, cookie/return-path policy, and session lifetimes. Account-only, partial Google, and incomplete account policy fail startup instead of inventing missing security settings.

Account mutations retain optimistic revisions and existing typed engine actions. `accountgame.Service` requires an authenticated user identity for create, list, read, action, elapsed-time synchronization, and single or bulk delete operations; owner predicates make another user's identifier indistinguishable from an absent game, and failed compare-and-swap writes return the current durable revision. Cookie authentication uses an opaque `Secure`, `HttpOnly`, `SameSite=Lax` verifier scoped to the configured application path. The current-account response exposes only an HMAC request-proof token bound to that verifier; every cookie-authenticated mutation requires it in `X-Sudoku-CSRF`, and configured origin policy remains exact.

SQLite foreign keys and unique constraints enforce issuer-subject identity, session verifier, game ownership, and claim fingerprint invariants. `guestclaim.Service.Claim` authenticates the sealed document before `db.DB.ClaimGuestGame` atomically creates the play run, account game, and stable claim record; same-owner retries return the original game, while cross-owner attempts reveal no owner details. Schema version 6 completes the development-stage replacement boundary: anonymous server sessions and API recovery records are removed rather than migrated or retained as compatibility scaffolding.

## Threat Model and Failure Policy

The design defends against forged guest state, OAuth login CSRF and code interception, session fixation and theft, cross-site mutations, insecure direct object references, duplicate claims, accidental identity linking by email, and sensitive logging. TLS termination and host security remain deployment responsibilities; compromised browsers, Google accounts, or hosts are outside the application boundary.

Authorization failures reveal no game existence across users. Database, sealing, or persistence failures never claim success; account state remains transactional, while guest state remains usable until the browser atomically replaces its local document.

## Acceptance Boundary

Package acceptance proves issuer-subject identity without email linking, digest-only session lookup, expiry and immediate revocation, owner-scoped reads and optimistic mutations, private-state cascades, and catalog preservation. `guestdoc.Sealer` provides AES-256-GCM sealed documents, versioned key selection, bounded decrypt-only rotation keys, expiry and payload validation, revision-checked engine actions, and active-key resealing without durable guest rows. `oidcauth.Manager` and `oidcauth.GoogleProvider` provide the bounded Google login transaction and verified issuer-subject profile result without storing Google tokens. `accountauth.Service` converts that verified result into one atomically rotated digest-only application session, reuses only the issuer-subject identity, and ignores unverified profile email. `guestclaim.Service` provides exactly-once same-owner claim and bounded cross-owner denial. `accountgame.Service` proves authenticated owner-scoped list, read, revisioned action, and delete behavior while hiding cross-owner identifiers. The versioned OpenAPI and runtime handler now expose the guest, Google login, current-account, session revocation, account deletion, account-game, and idempotent claim boundaries. The runtime integration test proves sealed guest replacement, login rotation, cookie authentication, HMAC-bound CSRF rejection, account-owned creation, exactly-once claim, account mutation, listing, and logout without external provider traffic.

Built-binary acceptance proves zero-config and configured guest startup, refresh-compatible replacement, key reuse across restart, platform-native private-key enforcement, tamper rejection, expiry/schema rejection, disabled account routes, and zero durable guest rows. Windows runs the same Go package tests plus direct zero-config restart and configured-startup smoke tests on a Windows host. Dedicated `privatefs` tests exercise protected DACLs, the allowed current-user/System/Administrators principals, inherited or additional principals, and reparse-point rejection; Unix tests independently exercise owner-only modes, links, and object types. Authentication tests use a deterministic local OIDC fixture to prove state, nonce, PKCE, issuer/audience, rotation, expiry, logout, and revocation without external network dependence.

Account acceptance proves exactly-once claim, cross-user denial, cross-browser resume, optimistic revision conflicts, game deletion, account deletion, and catalog preservation. Coordinated browser acceptance additionally proves the single-game IndexedDB lifecycle, login return path, explicit save action, My games, responsive accessibility, and independent keyboard, mouse, and touchscreen gameplay.
