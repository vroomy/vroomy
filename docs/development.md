# Developing Vroomy

Read [AGENTS.md](../AGENTS.md) for repository-wide agent instructions,
[STYLEGUIDE.md](../STYLEGUIDE.md) for coding conventions and adoption scope, and
[README.md](../README.md) for the runnable example. Documentation describes the
source in this checkout.

## Setup and checks

The module is `github.com/vroomy/vroomy`. Use the Go version required by
[`go.mod`](../go.mod) or newer; the [quickstart](../README.md#quickstart) names the
current minimum. Dependencies are pinned in `go.mod` / `go.sum`; inspect the
pinned apex source before documenting its APIs.

From the repository root:

```sh
go mod download
go test ./...
go vet ./...
go build ./...
```

`go mod download` needs network access unless dependencies are cached. Run `gofmt`
on edited Go files. For changes to shared state or lifecycle behavior, also run:

```sh
go test -race ./...
```

If the default build cache is not writable in an agent sandbox, use a writable
temporary cache with the same commands, for example:

```sh
GOCACHE="${TMPDIR:-/tmp}/vroomy-go-build" go test ./...
```

There is no Makefile, custom lint configuration, or checked-in CI workflow. Do not
assume additional checks have run remotely. `go build ./...` compiles all packages;
the root is a library, so `go run .` is not an application startup command.

Keep documentation versions aligned with `go.mod` when updating dependencies.
Keep a concrete minimum in the README and
link other version requirements to `go.mod` when a copied number adds no value.

## Source map

| Files | Responsibility |
| --- | --- |
| `vroomy.go` | Constructors, plugin initialization, route setup, listeners, shutdown, global `Register`. |
| `config.go`, `includeConfig.go` | TOML decoding, includes, process environment merge, TLS config. |
| `plugin.go`, `baseplugin.go`, `plugins.go` | Plugin interface, no-op base implementation, global registry. |
| `dependencies.go` | Tagged dependency discovery, validation, load ordering. |
| `utils.go` | Handler parsing/reflection, dependency field assignment, directory setup, host policy. |
| `route.go`, `routegroup.go` | Route and group schemas. |
| `environment.go` | String environment and typed accessors. |
| `response.go`, `flag.go` | Legacy response helpers and flag schema; not wired into routing/flag parsing. |
| `examples/hello/main.go`, `examples/hello/plugins/hello.go`, `config.example.toml` | HTTP quickstart with a separate plugin file that registers from `init()` when blank-imported by main. |

## Test coverage and manual verification

Measure library statement coverage and inspect uncovered lines with:

```sh
go test -coverprofile="${TMPDIR:-/tmp}/vroomy-coverage.out" .
go tool cover -func="${TMPDIR:-/tmp}/vroomy-coverage.out"
go tool cover -html="${TMPDIR:-/tmp}/vroomy-coverage.out"
```

These commands measure the root library package. Examples have no automated
tests; use the manual HTTP check below. Coverage measures executed statements,
not every possible branch or input. Remaining gaps include guards for malformed
process environment entries, a repeated nil-embedding check, and the constructor's
dependency-order callback failure. The removed-working-directory test skips on
platforms that can still resolve that directory. Live ACME issuance/renewal
remains a deployment check.

`environment_test.go` checks typed getters and required keys.
`config_test.go` checks environment values containing `=`, empty values, and
main/include/process environment precedence and host-policy fallback. The
`includeConfig_test.go` cases cover autocert directory/host merge behavior.
`dependencies_test.go` checks
dependency discovery and graph validation, including cycle membership without
assuming map order, and regression cases for its error comparator.
`utils_test.go` checks direct/factory handler parsing, literal argument semantics,
and missing parentheses. `reflection_test.go` covers registration shape,
backend assignment, and exported error contracts. `vroomy_test.go` checks graph
validation before Init, failed startup rollback, close order, registry reuse,
working-directory restoration, and filesystem causes. `startup_coverage_test.go`
adds file-based construction, data directory defaults, invalid groups and
dependencies, deferred group initialization, and rollback restoration failures.
`plugins_test.go` checks registry snapshots, invalid keys, closed-state contracts,
and aggregation of multiple cleanup failures. `handler_resolution_test.go`
checks direct/factory execution, literal arguments, configured middleware,
host-policy errors, and panic recovery with stack logging. `response_test.go`
and `route_test.go` cover the legacy response constructors and route display.

The exported `Plugins.Test()` and `Plugins.TestAsync()` both return
`"testing has not yet been implemented"`. They are unrelated to `go test`.

Use the example for a real HTTP check:

```sh
go run ./examples/hello config.example.toml
```

In another terminal:

```sh
curl --fail http://localhost:8080/
curl --fail http://localhost:8080/api/ping
```

Both return `PONG` followed by a newline. Stop the example with Ctrl-C. Change
`port` in a copy of the config if 8080 is occupied; pass that filename as the example's
first argument. The example's default config path is `config.example.toml`, relative
to the launch directory. It does not use `CONFIG_PATH`.

`handlerGroup_test.go` exercises configured sibling routes and nested groups over
HTTP for every configured method, full/spare middleware capacity, caller mutation,
and port release after explicit close.

`listener_test.go` covers startup failure, immediate cancellation, explicit close,
redirect methods and URL preservation, socket release, TLS HTTP/2, h2c activation
from the process environment, cached autocert certificates, hostname rejection,
and shutdown error causes. These tests bind temporary local ports and need a
sandbox that permits local listeners. Live ACME issuance/renewal is a deployment
check; tests do not contact a certificate authority.

`redirectListener_test.go` adds redirect response write failures, occupied ports,
repeated startup, close before startup, and HTTP/2 redirects.

When testing config loading, assert precedence and path resolution explicitly.
For service/plugin tests, avoid `t.Parallel` around the global registry or process
working directory. Restore any state you replace. An external test application in
a fresh subprocess can isolate registration and shutdown from other tests.

## Serving benchmarks

See [the serving allocation review](../benchmarks/serving/README.md) for
reproducible benchmarks, CPU/allocation profiles, before/after samples, and
remaining costs in apex and Go's HTTP transport. The benchmarks isolate warmed
handler dispatch from startup and also measure a loopback HTTP keep-alive request.
`handler_allocations_test.go` protects allocation-free direct dispatch and checks
receiver binding and concurrent requests.

## Known implementation gaps

These are current limitations to account for, not features promised by the docs.
Runtime fixes should include focused regression coverage and update the relevant
guide.

| Area | Evidence and effect |
| --- | --- |
| Static targets | `initRoute` in `vroomy.go` never adds a file handler for `Route.Target`. The former static-file quickstart could not serve its files. |
| Methods and flags | `initRoutes` falls back to GET for unrecognized methods (including HEAD/PATCH); group `Method` and dynamic flags have no active consumer. |
| Includes | `loadIncludes` iterates the original include slice. Newly appended nested entries are not loaded. Included `Plugins` and outer `Config.Plugins` are separate. |
| Handler/reflection validation | Missing closing handler parentheses return `ErrExpectedEndParen`, including a trailing `(`. Other malformed expressions are not fully validated; registration validates plugin shapes, anonymous pointers, and tagged fields; empty root paths return errors. Nil backends (including typed nil) and nil handler factory results are rejected. |
| Construction and reuse | `NewWithConfig` changes global working directory, retains and mutates config objects, and reuses registered plugin instances. One service exclusively owns the registry at a time. Failed construction closes attempted plugins in reverse dependency order and restores the previous directory. |

## Keeping documentation aligned

Update [configuration.md](configuration.md) and the example TOML for schema,
precedence, include, path, or TLS changes. Update [plugins.md](plugins.md) for
registration, dependency, lifecycle, and handler changes. Update this guide's gaps
when a fix lands.

Keep the README's first-run instructions independent of third-party plugins.
Compile complete Go examples, parse TOML examples, check local links, and smoke-test
changed startup instructions. Go doc comments are part of the documentation too:
avoid describing a legacy field or stub as an implemented feature.

## Listener dependency

Vroomy pins the published apex release
[`v0.16.1`](https://github.com/vroomy/apex/tree/v0.16.1)
for startup/Close synchronization, middleware ownership, and safe TLS walking.
Listener setup uses a helper with a deferred unlock; serving begins after it returns.
The previous v0.14.1 release has an immediate-cancellation race that can leave a listener open and make
shutdown wait forever.

The published module resolves through Go's normal proxy and checksum verification.
Both modules' race suites, vet and build, and the hello example have been checked.
No local `replace` directive or workspace file is required.
