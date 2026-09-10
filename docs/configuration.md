# Configuration reference

[`Config`](../config.go), [`IncludeConfig`](../includeConfig.go),
[`Route`](../route.go), and [`RouteGroup`](../routegroup.go) define the TOML schema.
Start with [config.example.toml](../config.example.toml), which works with
[`examples/hello`](../examples/hello/main.go).

## Loading and paths

`vroomy.New(filename)` calls `NewConfig(filename)`, supplies `env.dataDir = "data"`
only if the key is absent, and calls `NewWithConfig`.

`NewConfig` decodes TOML, loads includes, defaults an empty `dir` to `"./"`, creates
the environment map if needed, and adds missing process environment entries. It
does not start a server, change directories, or add the `dataDir` default.

`NewWithConfig` uses the supplied config directly. It calls `os.Chdir(cfg.Dir)`,
initializes the configured data directory, initializes/loads plugins, and creates
groups and routes. It does not parse files, load includes, copy the config, import
process environment entries, or supply defaults. A programmatic config should set
`Dir: "."`; an empty directory makes construction fail.

The configuration filename and all includes are resolved against the process
working directory **before** `dir` is applied. They are not relative to the
configuration file. `dir` changes the process working directory for all subsequent
code; relative `dataDir`, TLS paths, and plugin file accesses use that directory.

`Config.GetFilepath()` returns `CONFIG_PATH/config.toml` when the process environment
contains `CONFIG_PATH`, otherwise `config.toml`. `CONFIG_PATH` names a directory.
Applications can call this helper to choose their config path. `New("other.toml")`
uses the explicit filename and does not consult it.

## Top-level fields

| TOML key | Type / default | Current behavior |
| --- | --- | --- |
| `name` | string / empty | Decoded metadata; not used by the runtime. |
| `dir` | string / `"./"` via `NewConfig` | Working directory applied by `NewWithConfig`. |
| `port` | uint16 / `0` | HTTP port; `0` disables HTTP. |
| `tlsPort` | uint16 / `0` | HTTPS port; `0` disables HTTPS. |
| `tlsDir` | string / empty | Directory of certificate/key pairs; takes precedence over autocert. |
| `allowNonTLS` | bool / `false` | When HTTPS is enabled, `false` makes the HTTP listener redirect to HTTPS. `true` serves the routes over HTTP too. |
| `autoCertDir` | string / empty | Autocert cache directory; requires nonempty `autoCertHosts`. |
| `autoCertHosts` | string array / empty | Autocert host allowlist. |
| `include` | string array / empty | TOML files or directories loaded in order; see merge rules below. |
| `plugins` | string array / empty | Decoded import-path metadata with no active runtime consumer. The library runs all registered plugins regardless of this list; applications must import their plugin packages. |
| `[env]` | string-to-string table | Values supplied to every plugin's `Init` and `Load`. |
| `[[group]]` | array of tables | Named route groups, in initialization order. |
| `[[route]]` | array of tables | Routes, registered after all groups. |
| `[[flag]]` | array of tables | Legacy entries with `name`, `defaultValue`, `usage`; parsed but not applied. |

`Config.ErrorLogger func(error)` and `Config.Flags map[string]string` are Go-only
fields (`toml:"-"`). `ErrorLogger` is passed to httpserve's request error callback;
constructor/listener errors are still returned, and plugin/panic logs use the
standard logger. `Flags` has no active consumer. There are no built-in `-dataDir`,
`-d`, or `require` flags, and there is no automatic `testData` override.

Set at least one port to start a listener. With both ports zero, `Listen` still
blocks until its context ends even though no socket is opened.

Unknown TOML keys are not rejected: the decoder's undecoded-key metadata is not
checked. A typo can therefore be silently ignored.

## Environment and data directory

All `[env]` values must be strings, including numeric and boolean-like values:

```toml
[env]
dataDir = "./data"
workers = "4"
enabled = "true"
```

Precedence, from highest to lowest, is later includes, earlier includes, the main
file, then process environment. Keys are case sensitive. `DATA_DIR` and `dataDir`
are different keys; Vroomy does not expand `${VARIABLE}` inside TOML strings.

`New` supplies `dataDir = "data"` if still absent. An explicit empty string prevents
directory creation. The initializer uses `os.Mkdir`, not `MkdirAll`, so parent
directories must already exist. Plugins decide how to use this value.

Known limitation: `populateFromOSEnv` splits on every `=` and keeps only the second
piece. A process value such as `TOKEN=abc=def` becomes `abc` in `Environment`.
Explicit TOML values containing `=` are preserved.

| API | Missing key | Present key |
| --- | --- | --- |
| `Get` | `""` | Original string. |
| `GetInt`, `GetInt64`, `GetFloat64` | Zero, nil error | Parsed value or parse error. Integers are decimal. |
| `GetTime`, `GetTimeInLocation` | Zero `time.Time`, nil error | Parsed using the supplied Go layout and optional location. |
| `Must` | Error | Original string, including an empty string. Does not panic. |
| `MustInt`, `MustInt64`, `MustFloat64`, `MustTime`, `MustTimeInLocation` | Error | Same parsing as the corresponding `Get` method. |

There is no `GetBool` or `MustBool`. Use `strconv.ParseBool` in your plugin if needed.
See [environment.go](../environment.go) and its [tests](../environment_test.go).

## Includes

Place `include = [...]` at the top level, before `[env]` or array tables. A path
ending in `.toml` is decoded as an `IncludeConfig`; every other path is treated as
a directory. Directories are traversed recursively in filename order. Non-TOML
files are not skipped: they are treated as directories and generally cause errors.

Included files support only the fields in `IncludeConfig`, not `dir`, `port`,
`tlsPort`, `tlsDir`, `allowNonTLS`, or `name`.

- Environment keys overwrite earlier values.
- Groups, routes, flag entries, includes, and included plugin lists are appended.
- A nonempty included `autoCertDir` replaces both the cache directory and hosts.
  Hosts alone do not override earlier autocert settings.
- Nested `include` entries are appended but are not loaded during the same pass.
  List all required paths in the main file, or use a directory include.
- Included `plugins` populate `IncludeConfig.Plugins`, which is separate from the
  outer `Config.Plugins` field. Neither field filters or loads runtime plugins.

Main-file groups/routes precede included groups/routes. The combined ordering must
still put parent groups before children. Avoid duplicate group names: lookup
returns the first match.

## Routes and groups

| Field | Route | Group |
| --- | --- | --- |
| `name` | Optional description. | Name used by a child's `group` field. |
| `group` | Parent group name; empty uses the root. | Parent group name; empty uses the root. |
| `httpPath` | Route path appended to the group's prefix. | Path prefix appended to the parent's prefix. |
| `method` | Case-insensitive PUT, POST, DELETE, OPTIONS; all other values use GET. | Decoded but ignored. |
| `handlers` | Ordered plugin handler references. | Handlers inherited by descendant groups/routes. |
| `target` | Decoded but has no serving behavior. | Not a group field. |

For example, with the `hello` plugin registered by the example application:

```toml
[[group]]
name = "api"
httpPath = "/api"

[[group]]
name = "v1"
group = "api"
httpPath = "/v1"

[[route]]
group = "v1"
method = "GET"
httpPath = "/ping"
handlers = ["hello.Ping"]
```

This registers `/api/v1/ping`. Declare parent groups before their children; missing
parents and parents that have not been initialized cause construction errors.

httpserve v0.13.0 handles paths, parameters such as `/:id`, and terminal wildcards
such as `/*`. Handlers run in group-to-route order until the context is completed.
Write a response from a handler; a `target` alone will not serve a file. See the
[plugin guide](plugins.md) for exact method signatures and argument parsing.

`HTTPHandlers` can be supplied directly in Go and run before the same route/group's
string-resolved handlers. These slices and `RouteGroup.G` are mutated during
initialization. Use fresh configs when constructing services; reusing them can
append handlers again. `RouteGroup.G` is internal runtime state, not TOML input.

## TLS

For existing certificates, set `tlsPort` and `tlsDir`. The pinned httpserve v0.13.0
loads PEM certificate/key pairs sharing a pathname stem, such as `server.crt` and
`server.key`; keep certificate files in that directory. HTTP redirects to the
configured HTTPS port unless `allowNonTLS = true`.

For automatic certificates, omit `tlsDir` and set both `autoCertDir` and
`autoCertHosts`. The listener uses httpserve's autocert integration and automatically
accepts the ACME terms. Certificate issuance requires a reachable deployment
appropriate for the ACME challenge; the local HTTP quickstart does not exercise it.

A registered plugin with key `autocert` may export
`HostPolicy(context.Context, string) error`. If present, that policy is tried first;
the configured hosts remain a fallback allowlist even when the plugin rejects a
host. If the plugin is absent, only the configured host allowlist is used. A
registered `autocert` plugin with a missing or incorrectly typed method causes an
error when setting up autocert.

An enabled TLS port without either certificate source produces
`ErrInvalidTLSDirectory` during `Listen`, not during construction. Shutdown has a
separate HTTP-redirect-listener limitation described in
[development.md](development.md#known-implementation-gaps).
