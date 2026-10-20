# Writing plugins

A plugin is a Go value registered with `vroomy.Register`. It is compiled into the
application; Vroomy does not use Go's shared-object `plugin` package or discover
modules at runtime. Keep application plugins in their own files under `./plugins`,
with each file calling `vroomy.Register` from Go's `init()` function. The application
blank-imports the package to register the plugins before `main` runs, as shown by the
[hello example](../examples/hello/main.go).

## Application layout and registration

The runnable example uses this layout:

```text
examples/hello/
├── main.go
└── plugins/
    └── hello.go
```

The [plugin file](../examples/hello/plugins/hello.go) contains the `helloPlugin`
type, its methods, and its own `init()` registration. The main file imports it for
that registration side effect:

```go
import _ "github.com/vroomy/vroomy/examples/hello/plugins"
```

For your own application, use its module path, such as `example.com/myapp/plugins`.
Additional files such as `plugins/store.go` can use the same `plugins` package and
register their own instances from their own `init()` functions. Do not depend on
the order of these registrations; Vroomy resolves dependencies when it constructs
the service.

The `plugins` directory is an application convention; Vroomy does not scan it.
Only imported packages run their `init()` functions. If you use separate plugin
subpackages, import each one you need; importing a parent does not automatically
import its children.

Go's `init()` and Vroomy's `Plugin.Init` are different steps. Use `init()` to register
an instance, and the `Init` / `Load` hooks below to configure libraries and use
dependencies. An `init()` function cannot return an error; these examples panic on
duplicate registration, which indicates a programming error in the application.

## Preferred architecture

**As a best practice, plugins load libraries and provide handlers involving those
libraries when needed. This is recommended, not mandatory.** Most of a request's
logic should usually live in the library. A plugin handler translates between the
HTTP request/response and the library's API.

| Component | Preferred responsibilities |
| --- | --- |
| Plugin lifecycle | Read configuration, initialize the library, connect dependencies, expose a shared backend when useful, and close owned resources. |
| Plugin handler | Read and decode request input, call the library, and translate its result or error into HTTP status, headers, and output. |
| Library | Perform business logic, domain validation, persistence, and the main processing for the request. Return results and errors for the caller to use. |

For example, an order-creation request would typically follow this flow:

1. The plugin handler decodes the request into the input expected by the orders
   library.
2. The library applies order rules, performs the operation, and returns a result or
   error.
3. The plugin handler maps that result or error to the intended HTTP response.

Prefer library APIs that work with ordinary Go values so their behavior can be
tested and reused without an HTTP request or `*apex.Context`. Test business
behavior in the library and request decoding/response mapping in the plugin.

Use `Init` for setup that does not need injected dependencies and `Load` for setup
that does. `Backend()` can expose the library instance to other plugins without an
HTTP round trip. A plugin that only loads a library or shares a backend needs no
handlers.

The library may be a package in the same module or a separate module. Small handlers
and special cases can keep logic in the plugin when that is clearer; this pattern
does not require a new library for every trivial operation. See the
[style guide](../STYLEGUIDE.md#plugin-and-library-responsibilities) for the shared
guidance.

## Interface and lifecycle

Every plugin implements [`Plugin`](../plugin.go):

```go
type Plugin interface {
	Init(env Environment) error
	Load(env Environment) error
	Backend() any
	Close() error
}
```

Embedding `vroomy.BasePlugin` by value supplies no-op `Init`, `Load`, and `Close`
methods and a `Backend` method that returns nil. Override only what you need.
Register a non-nil pointer to a struct, using a unique nonempty key without `.`, `(`, or `)`; handler
references split at the first dot. Duplicate registration returns an error.

During `New` / `NewWithConfig`:

1. The service exclusively acquires a snapshot of the global registry and validates
   plugin fields and the dependency graph (missing providers, self-dependencies,
   and cycles).
2. The process changes to `Config.Dir` and initializes the data directory.
3. Every plugin's `Init(env)` runs before any dependency injection. Providers
   initialize before consumers; independent plugins have no stable order.
4. In dependency order, the service assigns tagged fields from provider `Backend()`
   values, then calls the consumer's `Load(env)`. Providers finish loading first.
5. Groups and routes resolve their handler methods/factories.

Use `Init` for your own environment/configuration and `Load` for setup requiring
other plugins. All plugins receive the same environment map. Avoid mutating it
as a way to communicate between plugins because initialization order is unstable.

`Close` closes the HTTP server and closes plugins in reverse dependency order.
Construction validates the dependency graph before `Init`. On later failure it
closes every plugin whose `Init` was attempted, including the failing plugin, and
restores the previous working directory. `Close` must tolerate partial `Init`.

The registry gives one service exclusive ownership of its instances. A second
constructor, registration, or registry `Close` while that service is active returns
`ErrPluginInUse`. Service `Close` releases ownership after plugin cleanup. Reuse
calls `Init` and `Load` again on the same instances, so plugins must support that
cycle; `New` does not allocate replacement instances. Do not mutate registered
plugins or configuration fields read at runtime while a service is active. Handler
registration copies its own slice storage, as described below.

## A complete plugin package

Save this example in `plugins/hello.go` in an application module named
`example.com/myapp`, and blank-import `example.com/myapp/plugins` from main. It
extends the runnable example's ping plugin with a handler factory. It has no
substantial library logic to delegate.

```go
package plugins

import (
	"fmt"

	"github.com/vroomy/apex"
	"github.com/vroomy/vroomy"
)

func init() {
	if err := vroomy.Register("hello", &helloPlugin{}); err != nil {
		panic(err)
	}
}

type helloPlugin struct {
	vroomy.BasePlugin
}

// Ping responds with PONG.
func (p *helloPlugin) Ping(ctx *apex.Context) {
	ctx.WriteString(200, "text/plain", "PONG\n")
}

// Message builds a handler that responds with its single configured argument.
func (p *helloPlugin) Message(args ...string) (apex.Handler, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Message expects one argument, received %d", len(args))
	}

	message := args[0]
	return func(ctx *apex.Context) {
		ctx.WriteString(200, "text/plain", message)
	}, nil
}
```

Configure routes using registration keys, not import paths:

```toml
[[route]]
httpPath = "/ping"
handlers = ["hello.Ping"]

[[route]]
httpPath = "/welcome"
handlers = ["hello.Message(Welcome)"]
```

The [configuration reference](configuration.md) explains the surrounding file.
The `plugins = [...]` list has no active runtime consumer; the application must
import/register its own plugins.

## Handler signatures and arguments

The resolver accepts exactly these exported method signatures:

```go
func (p *helloPlugin) Ping(ctx *apex.Context)
func (p *helloPlugin) Message(args ...string) (apex.Handler, error)
```

Direct handlers run for each request. Factories run once during route/group
initialization and return the per-request handler. Return an error from a factory
to reject invalid configuration. Returning `*vroomy.Response`, taking a standard
`http.ResponseWriter`, or using a different factory signature is not supported.

Direct handlers use the unbound reflected method and the registered receiver to
avoid the per-call allocation of a reflected bound function. Factories return
ordinary Go functions and avoid reflection while serving. Both paths can serve
without allocating in the warmed handler dispatch; response encoding and the
HTTP transport have separate costs. See the [serving benchmarks](../serving_benchmark_test.go).

- `"hello.Ping"` resolves the exported method on the registered `hello` instance.
- `"hello.Message(one,two)"` supplies two literal strings to a factory.
- `"hello.Message"` invokes a factory with zero arguments.
- `"hello.Message()"` supplies one empty string, not zero arguments.
- Parsing does not trim spaces, unquote values, expand environment variables, or
  support escaping commas/nested parentheses. `"hello.Message(one, two)"` passes
  `"one"` and `" two"`. Validate factory arguments explicitly.
- Parenthesized arguments on a direct handler are parsed but ignored. Omit them.

An opening parenthesis without a closing one, including a trailing `(`, returns
`ErrExpectedEndParen`. Other malformed expressions are not fully validated;
keep references in the documented forms.

Handlers in a group and route form a sequence. The pinned apex advances through
them automatically until the context completes; middleware can inspect/store
values and return to continue, or write a response to stop the sequence. There is
no `ctx.Next()` call. Use methods such as `ctx.WriteString`, `ctx.WriteJSON`,
`ctx.Param`, `ctx.Get`, and `ctx.Put` on `*apex.Context`.

The order is parent-group middleware, child-group middleware, then route handlers.
Vroomy copies handler storage for every route and nested group, preventing sibling
registrations from overwriting earlier sequences. Supplied handler slices may be
changed after construction without changing the registered sequence; handler
closures still share whatever application state they capture.

The root package's `Response`, `NewResponse`, and misspelled `NewAdopedtResponse`
remain exported data helpers, but the route resolver does not consume them. Use
apex's context response methods in new handlers.

## Dependency injection

Tag an exported field with the provider's registration key. A provider returns its
shared value from `Backend()`; a consumer receives that value before `Load`.
For example, these two files in the application's `plugins` package register a
provider and a consumer. Each plugin keeps its type, methods, and registration in
its own file.

`plugins/store.go` contains the provider and its small in-memory backend. An
application with substantial storage logic would normally load a library here:

```go
package plugins

import "github.com/vroomy/vroomy"

func init() {
	if err := vroomy.Register("store", &storePlugin{}); err != nil {
		panic(err)
	}
}

type memoryStore struct{}

// Greeting returns the stored greeting.
func (s *memoryStore) Greeting() string { return "hello" }

type storePlugin struct {
	vroomy.BasePlugin
	store *memoryStore
}

// Init creates the backend before other plugins can depend on it.
func (p *storePlugin) Init(env vroomy.Environment) error {
	p.store = &memoryStore{}
	return nil
}

// Backend exposes the store to dependent plugins.
func (p *storePlugin) Backend() any { return p.store }
```

`plugins/consumer.go` contains the consumer and the interface it needs:

```go
package plugins

import (
	"fmt"

	"github.com/vroomy/vroomy"
)

func init() {
	if err := vroomy.Register("consumer", &consumerPlugin{}); err != nil {
		panic(err)
	}
}

// Store supplies greeting data to the consumer.
type Store interface {
	Greeting() string
}

type consumerPlugin struct {
	vroomy.BasePlugin
	Store Store `vroomy:"store"`
}

// Load uses the store after dependency injection.
func (p *consumerPlugin) Load(env vroomy.Environment) error {
	if p.Store.Greeting() != "hello" {
		return fmt.Errorf("unexpected store greeting")
	}

	return nil
}
```

Blank-importing `example.com/myapp/plugins` registers both instances. Their Go
`init()` functions only register them; Vroomy injects the store before calling the
consumer's `Load`.

The receiving field's type must exactly match the returned concrete type, or be an
interface that the concrete type implements. There is no automatic conversion or
pointer dereferencing. `Backend()` must supply a usable non-nil value; inheriting
`BasePlugin.Backend()` is sufficient only when no consumer depends on that plugin.

Dependency scanning descends into anonymous embedded structs. Prefer value
embedding and exported tagged fields: reflection is not a general object graph
injector, and nil embedded pointers or inaccessible fields can panic. Use one field
per dependency key in a plugin; repeated tags overwrite each other in the dependency
map. See [dependencies.go](../dependencies.go) and [utils.go](../utils.go).

## Validation and error causes

Registration checks pointer/struct shape and nil anonymous fields. Tagged fields
must be exported and settable. Duplicate tags for one provider are rejected; use
one shared field. Recursive anonymous embedding is rejected, and anonymous scalar
fields without tags are ignored. Tagged anonymous fields receive a backend rather
than being traversed. Dependency backends must be non-nil, including typed-nil
pointers, maps, slices, functions, and channels. Assignment accepts exact types or
implemented interfaces. Handler factories must return a non-nil handler on success.

Missing registrations wrap `ErrPluginNotLoaded`; duplicate keys wrap
`ErrPluginKeyExists`. Use standard-library `errors.Is` for these and for underlying
plugin, filesystem, or handler-factory failures. Shutdown and constructor rollback
preserve multiple causes using `errors.Join`.
