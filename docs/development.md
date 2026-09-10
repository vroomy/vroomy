# Developing Vroomy

Read [README.md](../README.md) for the runnable example. Documentation describes the
source in this checkout.

## Setup and checks

The module is `github.com/vroomy/vroomy`, with Go 1.25.0 declared in `go.mod`.
Use that version or newer. Dependencies are pinned in `go.mod` / `go.sum`, including
httpserve v0.13.0; inspect that dependency's source before documenting its APIs.

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

`environment_test.go` checks typed getters and required keys.
`dependencies_test.go` checks dependency discovery and graph validation.
`utils_test.go` has a handler-parser case. `plugins_test.go` is an empty placeholder.
The existing suite does not establish end-to-end correctness for config merging,
dependency assignment, listeners, shutdown, or TLS.

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

When testing config loading, assert precedence and path resolution explicitly.
For service/plugin tests, avoid `t.Parallel` around the global registry or process
working directory. Restore any state you replace. An external test application in
a fresh subprocess can isolate registration and shutdown from other tests.

## Known implementation gaps

These are current limitations to account for, not features promised by the docs.
Runtime fixes should include focused regression coverage and update the relevant
guide.

| Area | Evidence and effect |
| --- | --- |
| Static targets | `initRoute` in `vroomy.go` never adds a file handler for `Route.Target`. The former static-file quickstart could not serve its files. |
| Methods and flags | `initRoutes` falls back to GET for unrecognized methods (including HEAD/PATCH); group `Method` and dynamic flags have no active consumer. |
| Includes | `loadIncludes` iterates the original include slice. Newly appended nested entries are not loaded. Included `Plugins` and outer `Config.Plugins` are separate. |
| Process environment | `populateFromOSEnv` truncates values containing `=`. Config values take precedence over process values. |
| Handler/reflection validation | Malformed handler arguments, invalid plugin shapes, and nil embedded pointers can panic. Existing tests do not cover all these paths. |
| Construction and reuse | `NewWithConfig` changes global working directory, retains and mutates config objects, and reuses registered plugin instances. Failed construction does not roll back initialized plugins. |
| Cancellation | `Listen` returns on context cancellation without closing resources. Its 100 ms startup message is a timer, not a readiness guarantee. `ListenUntilSignal` closes after `Listen`, treats `context.Canceled` as non-error, and combines other listen/close errors. |
| Signals | `onClose` registers SIGINT, SIGTERM, SIGABRT, but not SIGQUIT. It does not unregister or stop waiting when the parent context ends. |
| HTTP redirect shutdown | When HTTPS redirects are enabled, `getHTTPListener` creates a separate httpserve `Upgrader` that `Vroomy.Close` does not retain/close. Do not assume all listeners are released when embedding this mode in a long-lived process. |
| Close ordering | Plugins close in map iteration order, not reverse dependency order. Repeated `Vroomy.Close` returns `errors.ErrIsClosed` from `github.com/gdbu/errors`. |

## Keeping documentation aligned

Update [configuration.md](configuration.md) and the example TOML for schema,
precedence, include, path, or TLS changes. Update [plugins.md](plugins.md) for
registration, dependency, lifecycle, and handler changes. Update this guide's gaps
when a fix lands.

Keep the README's first-run instructions independent of third-party plugins.
Compile complete Go examples, parse TOML examples, check local links, and smoke-test
changed startup instructions. Go doc comments are part of the documentation too:
avoid describing a legacy field or stub as an implemented feature.
