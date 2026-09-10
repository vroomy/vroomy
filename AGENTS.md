# Working in Vroomy

This file applies to the entire repository. Vroomy is a Go library for a plugin-based
HTTP/HTTPS server. The root package is not an executable; a runnable library example
lives in `examples/hello`. No CLI is included in this checkout.

## Start here

1. Read [README.md](README.md) for setup and a working example.
   Read [STYLEGUIDE.md](STYLEGUIDE.md) before writing Go code; it defines coding
   conventions and how to apply them to existing code. Use this file for workflow
   and runtime constraints, and the style guide for coding style.
2. Read [docs/development.md](docs/development.md) for the source map, checks, and
   known gaps. Read [docs/configuration.md](docs/configuration.md) and
   [docs/plugins.md](docs/plugins.md) before changing config or plugin behavior.
3. Inspect `git status --short` and the relevant diff before editing. Preserve
   existing user changes, including untracked files.
4. Treat executable code and the versions in `go.mod` as the source of current
   behavior. Distinguish implemented behavior from TODOs and legacy fields. Update
   the corresponding docs when behavior changes.

## Commands

Use Go 1.25.0 or newer, as declared in `go.mod`. From the repository root:

```sh
go test ./...
go vet ./...
go build ./...
```

Run `gofmt -w` on changed Go files. Use `go test -race ./...` when changing shared
state, dependency loading, or lifecycle code. Tests use the standard `testing`
package; there is no Makefile or checked-in CI workflow. Do not confuse the
unimplemented `Plugins.Test` / `TestAsync` methods with the Go test suite.

For an HTTP smoke check, run `go run ./examples/hello config.example.toml`, request
`http://localhost:8080/` and `http://localhost:8080/api/ping`, and stop with Ctrl-C.
Both endpoints return `PONG`. The example creates `./data` unless configured
otherwise. See the development guide for restricted cache environments.

## Preferred plugin design

Keep application plugins in their own files under `./plugins`. Each plugin file
calls `vroomy.Register` from Go's `init()`; the application blank-imports the package
so registration happens before `main`. Follow `examples/hello/plugins/hello.go` and
`examples/hello/main.go`. Go's `init()` registers the instance; Vroomy's `Init` and
`Load` hooks initialize it when configuration and dependencies are available.

Prefer plugins that load/configure libraries and optionally provide handlers for
them. Most request logic belongs in the library; handlers decode inbound input,
pass it to the library, and translate its results or errors into HTTP responses.
Plugins may expose a shared backend without any handlers. This is a best practice,
not a requirement to extract a library for every small handler. See
[STYLEGUIDE.md](STYLEGUIDE.md#plugin-and-library-responsibilities) and the
[plugin guide](docs/plugins.md#preferred-architecture).

## Constraints that affect implementation

- Plugins are registered globally with `Register(key, &plugin)`. Startup snapshots
  all registered instances; `Config.Plugins` does not filter or dynamically load
  them. Registration belongs before `New` / `NewWithConfig`.
- Startup calls every `Init`, validates dependencies, then injects tagged fields
  and calls `Load` in dependency order. Dependencies are unavailable during `Init`.
  Independent plugins have no stable order. Close order is also unspecified.
- Register non-nil pointers to structs. Dependency fields must be exported and
  settable, tagged `vroomy:"registered-key"`, and match the provider's `Backend()`
  type (or an interface it implements). Prefer embedding `BasePlugin` by value.
- `New` parses config/includes and fills defaults. `NewWithConfig` does neither;
  supply at least `Dir: "."` explicitly. Both paths change the process working
  directory and share plugin instances. Avoid parallel tests that mutate either.
- Config routes resolve exported plugin methods with the exact signatures in the
  plugin guide. A handler writes through `*httpserve.Context`; returning
  `*vroomy.Response` is not supported by the route resolver.
- `target` has no file-serving implementation. Methods other than PUT, POST,
  DELETE, and OPTIONS fall back to GET. Group `method` is ignored. Parent groups
  must precede children.
- Legacy flags (`-d`, `-dataDir`, `require`, `[[flag]]`) are not wired up. Included
  files do not recursively execute their own `include` entries. See the config
  reference before relying on merge or environment precedence.
- `Listen` blocks and does not close resources on cancellation. `ListenUntilSignal`
  closes after listening ends. Constructor failures do not roll back plugin
  initialization; check the lifecycle limitations before adding resource ownership.

## Presenting pull request titles and descriptions

When providing a PR title and description in a response, present the title as plain
text and the description inside a fenced code block labeled `markdown`. Format the
description using [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md).
The code fence is for presenting the description in the response; when creating or
updating the PR itself, submit the Markdown body without the surrounding fence.

## Scope and documentation upkeep

Keep runtime fixes distinct from documentation corrections. Report discovered
bugs with source references; do not silently change public behavior during a docs
task. Keep examples runnable and avoid depending on another repository's plugins.

Keep `README.md`, `config.example.toml`, examples, Go doc comments, and the relevant
guide aligned. Check relative links and run the documented example after changing
setup instructions. Report the checks actually run and any unverified behavior.
