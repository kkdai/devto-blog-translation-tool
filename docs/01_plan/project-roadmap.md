# Project Roadmap

**Last updated:** 2026-08-27
**Current phase:** Phase 1 — Engineering Quality
**Overall status:** In Progress

## Current Status

The tool works end to end: it fetches every Dev.to draft with pagination, skips
articles that already have English titles, translates the rest with Gemini, and
updates and publishes them in place.

What is missing is the engineering scaffolding around it. There is no test
suite, the CLI has no options, and a duplicate copy of the program lives in a
second Go module.

| Area | State |
|---|---|
| Core translation flow | Working |
| Test coverage | None — zero `_test.go` files in the repository |
| CI | Builds and runs `go test`, which passes vacuously |
| Lint | Not configured |
| CLI options | None — every run processes all drafts and publishes them |
| Code duplication | `devto-article-translator/` is a near-identical second module |

## Phases

Phases are ordered by dependency, not by appeal. Tests come first because every
later phase changes behavior that nothing currently verifies.

| Phase | Theme | Status | Estimate |
|---|---|---|---|
| 1 | Engineering Quality | In Progress | ~2 days |
| 2 | Architecture Consolidation | Not Started | ~2 days |
| 3 | Translation Capability | Not Started | ~3 days |
| 4 | CLI Experience | Not Started | ~2 days |

---

### Phase 1 — Engineering Quality

**Status:** In Progress · **Estimate:** ~2 days

Establish the safety net. Nothing in Phases 2–4 should be attempted before this
lands, because all of it changes behavior that is currently unverified.

- Unit tests for `internal/utils` — `IsEnglish` and the frontmatter handling are
  pure functions with real edge cases (mixed CJK and Latin, the 80% ratio
  boundary, frontmatter with no trailing newline, code-fence style blocks)
- Unit tests for `internal/devto` against an `httptest` server — pagination
  termination, non-200 handling, context cancellation
- Unit tests for `internal/translator` with fake clients — the skip path,
  the 128-rune title truncation, per-article failure isolation
- Fix the CI Go version mismatch (workflow pins 1.20, `go.mod` requires 1.24)
- Add `golangci-lint` and a `gofmt` gate to the workflow

**Done when:** `go test ./...` runs real assertions, CI is green, and lint
passes on a clean tree.

---

### Phase 2 — Architecture Consolidation

**Status:** Not Started · **Estimate:** ~2 days · **Depends on:** Phase 1

Pay down the structural debt before adding features on top of it.

- Eliminate the duplicated program in `devto-article-translator/scripts/translator/`.
  It is a separate Go module carrying its own drifted copy of every source file,
  so fixes have to be applied twice and the root `go build ./...` never sees it.
  Decide whether the skill should build from the root module or be generated
  from it.
- Build the Gemini client once instead of per call. `Translate` and
  `TranslateTitle` each construct a fresh `genai.NewClient`, so a run over 300
  articles opens 600 gRPC clients.
- Extract a translator interface so the Gemini backend is swappable and the
  translation service can be tested without network access.

**Done when:** one copy of the program exists, one Gemini client per run, and
`internal/translator` has no compile-time dependency on a concrete backend.

---

### Phase 3 — Translation Capability

**Status:** Not Started · **Estimate:** ~3 days · **Depends on:** Phase 2

Extend what the tool can translate and how safely it does it.

- Target languages beyond English — the prompt and the "is it already English?"
  check are both hardcoded to English today
- Dry-run mode that shows what would be translated and published without
  writing anything back to Dev.to
- Translate-without-publishing, so drafts can be reviewed before going live
- Retry with backoff on transient Gemini and Dev.to failures. A single failure
  currently drops the article for the whole run.
- Chunked translation for long articles, so a body that exceeds the token
  budget is split rather than reported as truncated

**Done when:** a run can target a non-English language, be previewed safely, and
survive a transient API failure.

---

### Phase 4 — CLI Experience

**Status:** Not Started · **Estimate:** ~2 days · **Depends on:** Phase 3

Make the tool controllable. Today the only input is a yes/no prompt.

- Flags: `--dry-run`, `--limit`, `--article-id`, `--target-lang`, `--publish`
- Config file for defaults, so flags are not retyped every run
- Concurrent translation with a bounded worker pool and rate limiting
- Structured logging via `log/slog`, replacing the current `fmt.Println` calls,
  with the emoji progress output kept behind a human-readable handler

**Done when:** a single article can be translated by ID in dry-run mode without
editing source code.

## Non-Goals

Explicitly out of scope, to keep the tool small:

- A web UI or hosted service — this stays a CLI
- Platforms other than Dev.to
- Translation providers other than Gemini (Phase 2's interface makes one
  possible, but adding one is not planned)
- Editing or rewriting article content beyond translation

## Conventions

Detailed task breakdowns belong in sprint documents under `docs/02_implement/`,
not in this file. This roadmap tracks phase-level status only.

Status values: Not Started · In Progress · Completed · Blocked
