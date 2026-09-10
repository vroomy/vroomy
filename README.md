# Vroomy

<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-3-orange.svg?style=flat-square)](#contributors-)
<!-- ALL-CONTRIBUTORS-BADGE:END -->

![Vroomy billboard](vroomy-billboard.png)

Vroomy is a Go library for building HTTP/HTTPS services from registered plugins.
Plugins provide request handlers and shared backends; TOML config defines routes,
groups, environment values, and listeners. Plugins are compiled into your application.

## Quickstart

Use Go **1.25.0 or newer** (see [go.mod](go.mod)). From a checkout of this repository:

```sh
go mod download
go run ./examples/hello config.example.toml
```

In another terminal:

```sh
curl --fail http://localhost:8080/
curl --fail http://localhost:8080/api/ping
```

Both return `PONG` followed by a newline. Stop the server with Ctrl-C. The example
creates `./data`; copy the config and change `port` if 8080 is already in use.

The example keeps each plugin in its own file under the application's `plugins`
directory:

```text
examples/hello/
├── main.go
└── plugins/
    └── hello.go
```

[main.go](examples/hello/main.go) blank-imports the `plugins` package, which runs
the `init()` function in [plugins/hello.go](examples/hello/plugins/hello.go). That
function calls `vroomy.Register("hello", &helloPlugin{})` before `main` starts.
The application then loads [config.example.toml](config.example.toml). It needs no
external plugins or certificates. A minimal configuration for that plugin is:

```toml
dir = "."
port = 8080

[env]
dataDir = "./data"

[[route]]
httpPath = "/"
handlers = ["hello.Ping"]
```

## Use Vroomy in your application

From your own Go module:

```sh
go get github.com/vroomy/vroomy
```

Keep application plugins in their own files under `./plugins`, with each file
registering its plugin from `init()`. Blank-import the package in your main file:

```go
// For an application whose go.mod declares module example.com/myapp:
import _ "example.com/myapp/plugins"
```

Use your application's module path, not a relative Go import such as `"./plugins"`.
The import triggers registration before `main` runs. Then construct and run the
service:

```go
var (
	svc *vroomy.Vroomy
	err error
)

if svc, err = vroomy.New("./config.toml"); err != nil {
	log.Fatal(err)
}

if err = svc.ListenUntilSignal(context.Background()); err != nil {
	log.Fatal(err)
}
```

This snippet belongs in `main` with imports for `context`, `log`, and
`github.com/vroomy/vroomy`. See the [complete example](examples/hello/main.go) and
[plugin guide](docs/plugins.md) for registration and handlers.

`New` loads config, changes the process working directory to `dir`, initializes
plugins and dependencies, and registers routes. `ListenUntilSignal` starts the
configured listeners and closes the service when listening ends, including on
SIGINT/SIGTERM. `NewWithConfig` accepts an in-memory config but does not apply the
file loader's defaults; set `Dir` explicitly.

## Documentation

- [Configuration reference](docs/configuration.md): all fields, defaults, paths,
  environment accessors, include precedence, routes, and TLS.
- [Plugin guide](docs/plugins.md): registration, lifecycle, handlers, middleware,
  and dependency injection with complete code examples.
- [Development guide](docs/development.md): source map, checks, test coverage,
  and known implementation gaps.

The current runtime does **not** serve files from a route's `target` field. Provide
a handler instead. Legacy `-dataDir` / `-d` flags are not implemented; configure
`[env].dataDir`. These limitations are detailed in the guides above.

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

Run `gofmt` on changed Go files. See the [development guide](docs/development.md)
for smoke checks and test isolation. The project license is in [LICENCE](LICENCE).

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tr>
    <td align="center"><a href="http://itsmontoya.com"><img src="https://avatars2.githubusercontent.com/u/928954?v=4" width="100px;" alt=""/><br /><sub><b>Josh</b></sub></a><br /><a href="https://github.com/vroomy/vroomy/commits?author=itsmontoya" title="Code">💻</a> <a href="https://github.com/vroomy/vroomy/commits?author=itsmontoya" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/dhalman"><img src="https://avatars3.githubusercontent.com/u/1349742?v=4" width="100px;" alt=""/><br /><sub><b>Derek Halman</b></sub></a><br /><a href="https://github.com/vroomy/vroomy/commits?author=dhalman" title="Code">💻</a></td>
    <td align="center"><a href="http://mattstay.com"><img src="https://avatars0.githubusercontent.com/u/414740?v=4" width="100px;" alt=""/><br /><sub><b>Matt Stay</b></sub></a><br /><a href="#design-matthew-stay" title="Design">🎨</a></td>
  </tr>
</table>

<!-- markdownlint-enable -->
<!-- prettier-ignore-end -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!
