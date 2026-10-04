---
domain: Designs
status: Active
entry_points:
  - api/openapi.yaml
  - cmd/api.go
  - accountgame/accountgame.go
  - game/serialization.go
  - db/catalog.go
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

A guest game is one opaque sealed document stored by the browser. The sealed plaintext contains a random document identifier, exact puzzle presentation, complete versioned engine state and history, revision, actual difficulty, schema version, issued time, and expiry; authenticated encryption prevents browser forgery or authoritative edits.

An account game belongs to exactly one `users` row, references the shared base-puzzle catalog and a presentation-specific play run, and stores durable engine state. Every account-game read, mutation, list, and deletion resolves ownership from the authenticated web session rather than accepting a user identifier from the client.

A browser may retain at most one guest game. The backend therefore exposes no guest collection, guest list, guest recovery row, or bulk-import operation. Starting another guest game replaces the browser's prior guest document only after the new document is returned successfully.

## Identity and Session Model

`users` represents the application account. `external_identities` binds the normalized Google issuer and subject pair uniquely to one user; profile email and display fields are mutable attributes and never identity keys.

`web_sessions` stores a random verifier digest, user ID, creation and last-seen times, idle and absolute expiry, rotation lineage, and revocation time. Idle expiry defaults to 30 days and revocation takes effect immediately. The browser receives only an opaque `Secure`, `HttpOnly`, `SameSite=Lax` cookie with the narrowest application path; Google tokens and application bearer tokens never enter browser storage.

Google authentication uses authorization code with PKCE, state, and nonce. The backend creates short-lived, single-use login transactions, validates the exact issuer, audience, redirect URI, state, nonce, PKCE verifier, and token times, rotates any existing application session after callback, then returns the browser to an allowlisted relative destination.

Logout revokes the current web session. Account settings can revoke all web sessions and delete the account; account deletion revokes sessions and removes identities and owned games while preserving the shared puzzle catalog.

## Guest Document Boundary

Guest create and action endpoints accept no account game ID and never infer ownership mode from a cookie. Each action submits the latest sealed document, expected revision, and one existing engine action; the backend authenticates and opens the document, validates schema and expiry, applies the action through `game.Game`, increments revision once, and returns a replacement sealed document plus the authoritative snapshot.

Malformed, tampered, expired, or unsupported guest documents fail closed with stable recoverable errors. An older valid document may still be accepted because guest play has no competitive reward or server record; the claim boundary supplies replay protection where durable ownership begins.

The sealing key is a separately managed deployment secret with a versioned key identifier. Rotation retains bounded decrypt-only keys for the guest-document lifetime; logs and errors never include sealed documents, OAuth material, session cookies, puzzle history, or profile data.

## Explicit Claim Transaction

After sign-in, the browser may submit its one current guest document to the claim endpoint. One database transaction authenticates and validates the latest document, creates the owned account game and linked play-run state, records a unique claim fingerprint derived from the authenticated guest document, and returns the durable account game.

The claim fingerprint makes retries idempotent for the same user and prevents one guest document from creating duplicate durable games. A claim by another user fails without revealing the first owner. The browser deletes its guest record only after a successful claim response; cancellation or failure leaves the local copy intact.

Already authenticated players create account-owned games from the beginning. Account endpoints never accept sealed guest state except at claim, and guest endpoints never switch to account mode merely because a web-session cookie is present.

## API and Persistence Shape

The SQLite persistence foundation stores provider-keyed identities, verifier-digest web sessions, and owner-scoped account games in schema version 3. Account games link shared base puzzles to presentation-specific play runs, retain complete engine state with optimistic revisions, and disappear with their owning user while catalog rows and play-run history remain.

The OpenAPI contract names guest and account resources separately. Guest operations carry sealed documents; authentication operations start login, complete callback, report the current account, log out, and revoke sessions; account-game operations list summaries, create, read, mutate, delete, and claim one guest document. `oidcauth.Manager` owns short-lived single-use state, nonce, PKCE, return-path, issuer, audience, and token-time validation, while `oidcauth.GoogleProvider` owns discovery, code exchange, signature verification, and profile-claim extraction.

Account mutations retain optimistic revisions and existing typed engine actions. `accountgame.Service` requires an authenticated user identity for list, read, action, and delete operations; owner predicates make another user's identifier indistinguishable from an absent game, and failed compare-and-swap writes return the current durable revision. CSRF protection applies to cookie-authenticated mutations through same-origin enforcement plus a dedicated token or equivalent explicit request proof; login, callback, claim, and destructive account operations receive bounded rate limits.

SQLite foreign keys and unique constraints enforce issuer-subject identity, session verifier, game ownership, and claim fingerprint invariants. `guestclaim.Service.Claim` authenticates the sealed document before `db.DB.ClaimGuestGame` atomically creates the play run, account game, and stable claim record; same-owner retries return the original game, while cross-owner attempts reveal no owner details. The account schema is a development-stage replacement boundary: anonymous server sessions and recovery records may be discarded during cutover rather than migrated or retained as compatibility scaffolding.

## Threat Model and Failure Policy

The design defends against forged guest state, OAuth login CSRF and code interception, session fixation and theft, cross-site mutations, insecure direct object references, duplicate claims, accidental identity linking by email, and sensitive logging. TLS termination and host security remain deployment responsibilities; compromised browsers, Google accounts, or hosts are outside the application boundary.

Authorization failures reveal no game existence across users. Database, sealing, or persistence failures never claim success; account state remains transactional, while guest state remains usable until the browser atomically replaces its local document.

## Acceptance Boundary

Package acceptance proves issuer-subject identity without email linking, digest-only session lookup, expiry and immediate revocation, owner-scoped reads and optimistic mutations, private-state cascades, and catalog preservation. `guestdoc.Sealer` provides AES-256-GCM sealed documents, versioned key selection, bounded decrypt-only rotation keys, expiry and payload validation, revision-checked engine actions, and active-key resealing without durable guest rows. `oidcauth.Manager` and `oidcauth.GoogleProvider` provide the bounded Google login transaction and verified issuer-subject profile result without storing Google tokens. `accountauth.Service` converts that verified result into one atomically rotated digest-only application session, reuses only the issuer-subject identity, and ignores unverified profile email. `guestclaim.Service` provides exactly-once same-owner claim and bounded cross-owner denial. `accountgame.Service` proves authenticated owner-scoped list, read, revisioned action, and delete behavior while hiding cross-owner identifiers. Runtime account and guest routes remain unavailable until OpenAPI wiring adopts these boundaries.

Built-binary acceptance will prove guest start, action, completion, refresh-compatible replacement, tamper rejection, expiry/schema rejection, and zero durable guest rows when the routes become available. Authentication tests use a deterministic local OIDC fixture to prove state, nonce, PKCE, issuer/audience, rotation, expiry, logout, and revocation without external network dependence.

Account acceptance proves exactly-once claim, cross-user denial, cross-browser resume, optimistic revision conflicts, game deletion, account deletion, and catalog preservation. Coordinated browser acceptance additionally proves the single-game IndexedDB lifecycle, login return path, explicit save action, My games, responsive accessibility, and independent keyboard, mouse, and touchscreen gameplay.
