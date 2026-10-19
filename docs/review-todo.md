# Documentation review follow-up

Assessed on 9 October 2026 against commit `7a5406b` (also tag `v0.20.0`),
the supplied **Vroomy documentation review** dated 5 October 2026, and the cached
source of apex v0.14.1. The starting worktree was clean.

The review is substantially correct. The immediate work is version/documentation
alignment, a dependency routing bug, and a test assertion bug discovered during
this assessment. Existing style debt needs targeted migration; some cases need
additional guidance rather than a stricter blanket rule.

Status updated on 10 October 2026: **all 20 items completed and integrated**,
including lifecycle cleanup with the published apex dependency fix.
Checked items are implemented in the current worktree.
P1 means address before the next release; P2 means follow-up
work. Optional items are product or maintenance decisions. The assessment and
source line references below describe the original baseline unless stated otherwise.

Completed work upgrades the module minimum to Go 1.27.2, aligns the guides,
fixes environment truncation and the trailing-`(`
parser panic, repairs dependency assertions, migrates explicit returns/spacing,
removes result shadowing and obsolete commentary, and expands the style guide.
Plugin shape/dependency validation, exclusive service ownership, constructor
rollback, error causes, and handler slice ownership are now addressed.
Listener/signal cleanup and protocol regressions are integrated with the published
dependency fixes.

## P1: documentation corrections

- [x] **D1 — Align version requirements and prevent another split.**
  [go.mod](../go.mod) now requires Go **1.27.2** and pins the published apex
  fixes at **v0.14.2**. Correct
  [README.md](../README.md), [AGENTS.md](../AGENTS.md), and
  [development.md](development.md) for Go; correct the apex references in
  [configuration.md](configuration.md), [plugins.md](plugins.md), and development.
  Keep a concrete Go requirement in the quickstart; link other version references
  to the authoritative source where a number adds no value. Use `go.mod` as the
  source of truth when updating documentation alongside dependencies.
- [x] **D2 — Describe the HTTP redirect precisely.** Configuration currently
  overstates `allowNonTLS = false`: only GET redirects, with status 301; other
  methods receive 404. The URL includes the configured TLS port even for 443.
  Record the method limitation beside the setting and in the development gaps.
  Do not silently expand redirect methods while correcting prose.
- [x] **D3 — Qualify the group-handler contract pending R2.** Explain the spare
  slice-capacity bug in both configuration and plugin guides, including sibling
  nested groups. Preserve the intended parent-to-child-to-route contract and
  identify the current defect; remove the caveat after the fix is verified.
- [x] **D4 — Clarify route normalization and matching.** Document `path.Join`
  cleaning of grouped paths (`.`/`..` and trailing slashes), the leading slash
  requirement for root routes, named `/:id` capture through `ctx.Param`, and
  non-capturing `/*`. A wildcard returns success immediately when reached, so
  suffix segments after it are ignored. Recommend terminal wildcards, as the
  existing guide already does. Empty root paths also need validation in R3.
- [x] **D5 — Document listener defaults and HTTP/2 controls.** Record the dependency
  defaults: read/write 30 seconds, idle 90 seconds, and configured
  `MaxHeaderBytes` 4096. These are not TOML settings. Describe TLS HTTP/2
  advertisement and h2c activation through the **process** environment variable
  `USE_HTTP2_RAW`: its presence enables h2c, even if its value is empty or
  `"false"`. A TOML `[env]` entry alone does not set the process environment.
  Distinguish source inspection from tested protocol negotiation.

Dependency evidence for D2–D5: apex v0.14.1
[upgrader.go](https://github.com/vroomy/apex/blob/v0.14.1/upgrader.go)
(`NewUpgrader`, `upgradeConn`),
[group.go](https://github.com/vroomy/apex/blob/v0.14.1/group.go),
[route.go](https://github.com/vroomy/apex/blob/v0.14.1/route.go) (`newRoute`, `check`),
and [apex.go](https://github.com/vroomy/apex/blob/v0.14.1/apex.go)
(`defaultConfig`, the listener methods). Vroomy invokes the default-config
listener variants in [vroomy.go](../vroomy.go), `listenHTTP` and `listenHTTPS`.

## P1: correctness and regression work

- [x] **R1 — Repair the dependency-test comparator before relying on its results.**
  In [dependencies_test.go](../dependencies_test.go), line 374, `errorEqual`
  asserts `a` twice instead of asserting `b` on the second line. For an expected
  `*errors.ErrorList`, this compares the expected list with itself, ignoring the
  actual error. A temporary probe confirmed that an expected two-error list
  compares equal to nil. Correcting the helper in an overlay exposed obsolete
  cycle-error expectations in `Test_makeDependenciesMap`. Reconcile those
  assertions with the intended error contract; avoid depending on map iteration
  order. Verify that missing, different, and unexpected errors fail assertions.
- [x] **R2 — Fix group handler slice ownership at every composition boundary.**
  **Complete in Vroomy:** `handlerGroup` flattens paths and owns a copied chain
  at each route/group boundary before registering on the root Serve. Integration
  tests pass against the published dependency for sibling routes, every supported
  configured method, nested groups, caller mutation, and both full/spare capacity.
  The published upstream ownership fix is also pinned in `go.mod`.
  In the original dependency, `PUT`, `POST`, `DELETE`, `OPTIONS`, `Handle`, and `Group` append to `g.hs`
  without copying. If capacity remains, later registrations overwrite handlers
  retained by earlier routes/groups. GET copies its middleware first, but that
  does not protect a nested group's middleware from sibling-group construction.
  Fix ownership at each composition boundary and cover sibling routes, different
  methods, nested groups, and both spare/full capacity. Add a Vroomy integration
  regression for configured groups. A workaround only in Vroomy's top-level
  group creation needs proof that nested composition is safe too.
  Evidence: dependency `group.go`, lines 47, 60, 73, 86, 99, and 112;
  [vroomy.go](../vroomy.go), `initRouteGroup` and `initRoutes`.
- [x] **R3 — Return errors for malformed configuration instead of panicking.**
  **Complete:** registration/constructor validation covers plugin shape, tagged
  fields and anonymous pointers; nil backends (including typed nil) and empty root
  paths return errors. Exact and interface injection and nil factory results are covered.
  Guard empty handler argument substrings before indexing in
  [utils.go](../utils.go), `getHandlerParts`, line 52. Validate plugin shapes,
  tagged fields, nil embedded pointers, and empty route paths before reflection
  or dependency route construction. Cover these failures, exact/interface backend
  assignment, and existing valid handler forms. Define typed-nil backend handling
  explicitly. Preserve valid argument semantics unless deliberately changing them.
- [x] **R4 — Preserve process environment values containing equals signs.**
  Replace the unrestricted split in [config.go](../config.go),
  `populateFromOSEnv`, line 206, with a first-separator operation. Test empty
  values, multiple equals signs, and existing TOML/include precedence. Remove
  the truncation caveat only with the runtime fix.
- [x] **R5 — Define ownership and repair lifecycle cleanup.**
  **Complete:** listeners close on every exit, signal notification stops, and
  shutdown waits for listener goroutines. Exclusive ownership and reverse-order
  rollback/shutdown are covered by regressions.
  Retain and close redirect listeners; the original apex `Upgrader` exposes
  no `Close`, so this needs a dependency API change or replacement implementation. Stop signal
  notification and cancel its waiting goroutine when listening ends. Define
  constructor-failure cleanup and close order for shared registered instances
  before implementing rollback: blindly closing a shared plugin can break another
  service. Test failed startup, cancellation, shutdown, listener release, and
  multiple-service behavior; run the race suite with these regressions.
  Evidence: [vroomy.go](../vroomy.go), `NewWithConfig`, `getHTTPListener`,
  `ListenUntilSignal`, `Close`, and `onClose`.

## P2: coverage and style migration

- [x] **T1 — Add focused config/lifecycle contract tests.**
  **Complete:** includes/path/order/non-recursion,
  dependency injection, redirects/shutdown, TLS HTTP/2, h2c, and cached autocert
  certificates/hostname policy are covered. Live issuance remains a deployment check.
  Cover include precedence,
  path resolution, nested include non-recursion, autocert merge behavior and
  primary-policy/allowlist fallback, redirect methods, and listener cleanup.
  Decide deliberately which legacy behaviors to preserve and which to change;
  tests should not make every documented bug permanent. Isolate the registry and
  working directory using cleanup or subprocesses. Add new cases alongside R1–R5
  rather than postponing their regression coverage to this task.
- [x] **S1 — Migrate explicit returns and spacing in intentional changes.**
  Replace naked returns in affected value-returning functions, preserving all
  returned values, including partially populated results on error. Add required
  blank lines after completed control-flow statements. Do not restyle whole files
  during a small docs edit or unrelated bug fix.
- [x] **S2 — Remove local/named-result shadowing.** Use distinct primary/backup error
  names in host-policy fallback, and a distinct plugin-close error in `Close`.
  Preserve which failure is returned and how errors are aggregated.
- [x] **S3 — Review error contracts before improving wrapping.** Add operation/path
  context and preserve external causes; replace error-message classification with
  a typed/sentinel contract. Review compatibility separately from formatting.
  Existing exported legacy sentinels are not automatically the correct contract.
- [x] **S4 — Improve comments and remove misleading dead commentary.** Correct the
  claim that the 100 ms timer proves successful listening, clarify constructor
  and shutdown ownership/concurrency contracts, and remove or replace obsolete
  commented-out lifecycle/filter code. Keep future proposals outside descriptions
  of implemented behavior. Consolidate third-party import grouping in `plugins.go`.

### Original STYLEGUIDE.md findings

This table records the assessment baseline, not the final follow-up state. The
checked S/G items above resolve returns, spacing, shadowing, import grouping,
misleading comments, validation output, and supporting-type constructor placement.
Error contracts and resource ownership remain open.

| Finding | Concrete evidence | Assessment |
| --- | --- | --- |
| Naked returns with results | `environment.go:21`, `config.go:33`, `vroomy.go:48`, `utils.go:139`, `dependencies.go:20` | Clear violation of **No Naked Returns**; widespread in existing code. No exception is needed. |
| Shadowing named `err` | `vroomy.go:504` in `Close`; `config.go:193` in the host-policy closure | Clear violation of **Avoid Shadowing**. Backup failure deliberately preserves the primary failure; use distinct names to express that. |
| Lost external error causes | `vroomy.go:66`/`:71` use `%v`; `config.go:135` discards the `ReadDir` error entirely | Violates the external-operation error rule. Wrapping changes need contract review under the same guide. |
| Error classification by text | `utils.go:158`, `isUnregisteredPluginError` | Clear mismatch with typed/sentinel cause identification. Requires a deliberate error-contract change. |
| Missing post-block blank line | `vroomy.go:508` before the final return; `dependencies_test.go:453` before the final return | Clear formatting violation; `gofmt` does not enforce this rule. |
| Extra third-party import split | `plugins.go:8` and `:10` separate queue and errors | Inconsistent with grouping third-party imports together; small cleanup. |
| Resource ownership | `vroomy.go:79` onward lacks failed-init rollback; `:378` discards the redirect server reference; `:470`/`:532` lack cancel/signal cleanup | Conflicts with **Resource Ownership**. These are runtime changes, with shared-plugin ownership needing design first. |
| Misleading implementation comment | `vroomy.go:443` equates timer expiry with successful listening | Violates accurate behavior/contract comments; the development guide already correctly distinguishes notification from readiness. |
| Library output during validation | `dependencies.go:108`, `fmt.Println("Loading", key)`, also reached from `Validate` | Design/style concern: validation emits misleading progress output. The guide's logging boundary rule does not explicitly address successful-operation output. |
| Related internal map types in one file | `dependencies.go:56` and `:136`; `makeDependencyMap` is far above `dependencyMap` | Requires judgment under the supporting-type exception. Clarify the exception or separate cohesive types and place each constructor above its type. |

All production exported types/functions/methods/constants already have comments;
the issue is contract detail and some field-comment wording, not wholesale missing
Go docs. All checked Go files were already `gofmt` clean. Primary Vroomy methods
stay with their type and its constructors precede it.

## P2: clarify the style guide's use cases

- [x] **G1 — Document slice/map ownership at API boundaries.** When retaining,
  combining, or mutating caller-provided collections, document ownership and copy
  where independent storage is required. Include middleware composition as an
  example. This is the missing guidance most directly connected to R2.
- [x] **G2 — Explain ownership of framework-managed and shared resources.** Distinguish
  owned listeners/cancel functions from externally registered plugin instances;
  define failed acquisition/initialization cleanup. Also explain that context
  cancellation alone does not necessarily close sockets.
- [x] **G3 — Clarify registration panic versus runtime validation.** Duplicate
  registration in executable `init()` examples is a programming invariant failure.
  Malformed TOML/reflection input is an ordinary error to return. Make the existing
  examples' justified panic use explicit rather than weakening operational error rules.
- [x] **G4 — Clarify when tightly coupled private support types share a file.**
  State how constructors are placed in that case. Function-local test types already
  fall outside the production file-per-type rule and need no migration.
- [x] **G5 — Describe library output and framework error callbacks.** Validation
  should not unconditionally print progress. Explain when reporting through a
  configured callback is appropriate. `ctx.WriteString`/`WriteJSON` return no
  error and report write failures through apex's callback, so the example is
  not ignoring a returned error and needs no artificial wrapper.

Do not add exceptions for naked returns or result shadowing. Do not ban all `:=`:
short scoped lookups and the example's `configPath` declaration already fit the
guide. Unnamed trivial results and the small hello plugin also fit explicit
existing allowances; forcing named results or another library would overapply it.

## Review comments I qualify or disagree with

- **“Go 1.25 cannot build this module” needs a toolchain distinction.** An active
  1.25 toolchain cannot satisfy the minimum. An installed 1.25 `go` command can
  select/download a newer toolchain with `GOTOOLCHAIN=auto`. The documented minimum
  still needs correction. See [Go's toolchain documentation](https://go.dev/doc/toolchain).
- **Version drift is not evidence that runtime tests would prevent it.** Ordinary
  `go test` does not compare prose with `go.mod`. Update version claims alongside
  dependency changes; behavior tests address different gaps.
- **The dependency assertion coverage is weaker than the review suggests.** The
  empty plugin tests and limited parser tests are described correctly, but R1
  invalidates some existing dependency error assertions too.
- **The group bug is broader, and conditional.** Nested `Group` composition also
  aliases slices. Spare capacity is the trigger, not every grouped route. Our
  reproduction uses an explicit length-3/capacity-4 middleware slice. The frequency
  of spare capacity is not a dependable API guarantee or a reason to downrank it.
- **The wildcard text is incomplete rather than plainly incorrect.** The current
  configuration guide calls `/*` a terminal wildcard and calls `/:id` a parameter;
  it does not promise wildcard capture. Adding the matching details is useful,
  but these are dependency semantics, not automatically Vroomy defects.
- **An explicit `:443` is valid.** Document the URL formatting; omitting the default
  port is optional polish, whereas GET-only behavior and listener ownership matter.
- **Duplication is sometimes useful.** Keep startup/global-state hazards beside
  APIs and in agent constraints where readers act on them. Centralize frequently
  changing version numbers rather than eliminating contextual safety information.
- **A changelog, README release badge, and configurable timeout fields are optional.**
  They may improve maintenance or embedding, but their absence is not a documented
  contract violation. Document current listener defaults first. `doc.go` already
  mentions working-directory changes and the process-wide registry; it could link
  to fuller lifecycle details rather than repeat every guide.

## Initial review verification

- Read the supplied DOCX, repository guides, production Go sources/tests, and the
  relevant cached apex v0.14.1 source. Confirmed HEAD equals tag v0.20.0.
- `go test ./...`, `go vet ./...`, and `go build ./...` passed with Go 1.26.5 and
  a writable temporary build cache. Build emitted a non-fatal sandbox module-cache
  metadata warning. `gofmt -l` reported no files needing formatting.
- Built and ran the hello example with a temporary config changing only HTTP port
  and data directory. GET `/` and `/api/ping` both returned 200 with `PONG\n`;
  SIGINT produced exit status 0. Temporary data remained outside the repository.
- Temporary diagnostic tests reproduced R1 and R2 without modifying tracked Go
  files: a Go overlay for Vroomy and an unchanged temporary copy of apex with
  added probes. The correct-behavior assertions intentionally failed: the first
  non-GET route returned B instead of A; a nested GET route ran sibling B's
  middleware; the comparator accepted nil for an expected error list. GET sibling
  routes passed the same direct-route probe.
- Correcting only the comparator through an overlay made the existing cycle test
  fail on stale expected error text, confirming that the original passing suite
  masks a real mismatch.
- TLS/ACME issuance, actual HTTP/2 negotiation, redirect requests/shutdown, and
  race behavior were not exercised. Related findings are source-inspected;
  tests are proposed above. No runtime fix was applied during the initial assessment; follow-up fixes are
  recorded in the checked items above.

## Easy-win follow-up verification

- Verified Go 1.27.2 is the newest stable release in the
  [official release feed](https://go.dev/dl/?mode=json). Downloaded the Darwin ARM64
  toolchain to a temporary directory and verified its published SHA-256 before use.
  The module and README now require Go 1.27.2; other guides link to that minimum.
- On Go 1.27.2, `go test ./...`, `go test -race ./...`, `go vet ./...`, and
  `go build ./...` passed. Build emitted only the non-fatal sandbox module-cache
  metadata warning noted above. Changed Go files are `gofmt` clean.
- All 72 local links in the edited guides and this review document resolved;
  `git diff --check` passed.
- Rebuilt the hello example with Go 1.27.2 and the updated source. With a temporary
  config selecting an available port and temporary data directory, GET `/` and
  `/api/ping` both returned 200 with `PONG\n`; SIGINT produced exit status 0.
- New tests cover process environment preservation/precedence, autocert merge
  behavior and host-policy fallback, handler parser failures, and dependency
  assertion regressions. No TLS/ACME or HTTP/2 negotiation was exercised; the
  dependency routing defect and lifecycle integration gaps remain documented.

## Validation and ownership follow-up

- Registration rejects invalid shapes, keys, anonymous nil fields, unsettable
  dependencies, and duplicate provider tags. Dependency fields support anonymous
  structs and pointers; untagged scalar embedding is ignored.
- Construction validates the graph before acquiring plugin resources. The registry
  leases its instances to one service; concurrent construction/registration or
  registry shutdown returns `ErrPluginInUse`. Failure closes attempted plugins in
  reverse dependency order and restores the previous working directory.
- Filesystem and plugin failures retain causes. Missing and duplicate registration
  errors wrap their exported sentinels, replacing message-based classification.
- Added regressions for validation, injection, rollback, exclusive ownership,
  include paths/order/non-recursion, and cause preservation. Go test/race/vet/build
  passed against the published dependency with Go 1.27.2.

## Middleware ownership follow-up

- Configured groups use owned middleware chains, including nested siblings.
  Root registration preserves apex's response completion and hook behavior.
- HTTP integration regressions pass with the published apex v0.14.1 under
  Go 1.27.2, with and without spare middleware capacity. All five configured
  methods and caller-slice mutation are covered. Paths keep the existing cleaning
  behavior and empty resolved root paths return errors.
- Listener/signal cleanup and its protocol suite are integrated with the published
  upstream startup/Close race fix.

## Lifecycle follow-up

- `Listen` now owns cleanup on every exit and waits for listener goroutines.
  Redirect sockets are retained and closed. Signals use `signal.NotifyContext`
  with a deferred stop; normal cancellation does not hide joined cleanup errors.
- Upstream apex commits fix middleware ownership, the startup/Close race,
  redirect Close support, and TLS walk errors/short names. Existing router tests
  had stale zero-parameter assertions for named routes; corrected the assertions
  without changing matching behavior.
- Local HTTP/TLS/h2c/cached-autocert regressions and both modules' full race suites,
  vet and build pass with the dependency fixes. The published v0.14.1 control
  reproduced a data race and shutdown hang on immediate cancellation.
- Vroomy pins the published dependency commit with checksums verified through Go's
  normal module resolution. The integrated branch passes test/race/vet/build,
  documentation link checks, and the hello HTTP/SIGINT smoke test without
  a repository-specific replace or workspace file.
