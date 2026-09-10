# Go Style Guide

This guide defines the coding standards for Vroomy. Favor explicit control flow,
small responsibilities, and predictable structure so the next person can review
and maintain the code confidently.

Read [AGENTS.md](AGENTS.md) for the agent workflow and
[docs/development.md](docs/development.md) for architecture and verification.
This file defines style; executable code defines current runtime behavior.

## Scope and adoption

Apply this guide to new code and functions being substantially changed. A comment
edit or a one-line bug fix does not require restyling an entire function or file.
Keep broader style migrations in separate, intentional changes.

Existing code is not uniformly compliant, particularly around naked returns and
error wrapping. Its presence is not an exception for new code. Preserve public
APIs, error contracts, and behavior when improving style: changing error identity,
cleanup, or initialization order is a behavior change that needs its own review.

Rules stated as **must**, **do not**, or **never** are requirements within this
scope. **Prefer** identifies a default to apply with judgment. The narrow exceptions
below are part of the guide; they do not require a separate approval step.

## Project Structure

### File Per Type

Each primary production type gets its own file, with its constructors and all its
production methods. Small supporting types may stay with the primary type when
they exist only to support it.

Tests belong in corresponding `_test.go` files. Generated code and platform/build
constraints may require separate files; use those mechanisms when needed, rather
than splitting methods just to shorten a file. Match nearby file naming instead
of renaming unrelated files.

### Plugin Files and Registration

Keep application plugins in their own files under `./plugins`, for example
`plugins/hello.go` and `plugins/store.go`. Each file defines its plugin type and
methods and calls `vroomy.Register` from a package-level `init()` function.

For plugins in the same directory, use a shared `plugins` package and blank-import
it from the application's main package using its full module import path. Go runs
the registration functions before `main`. Separate plugin subpackages are also
possible; import each package that needs to register.

Keep Go's `init()` focused on registration. Initialize the underlying libraries in
Vroomy's `Init` or `Load` hooks, when configuration and dependencies are available.
See [the runnable example](examples/hello/main.go) and its
[plugin file](examples/hello/plugins/hello.go).

### Constructor Placement

Place constructors directly above the type they construct, followed by its methods.
Lifecycle methods such as `Init` and `Load` are methods, not constructors.

This keeps entry points predictable and related behavior together.

For example, in `counter.go`:

```go
// counter.go

// NewCounter constructs a Counter starting at initial.
func NewCounter(initial int) (c *Counter) {
	c = &Counter{value: initial}
	return c
}

// Counter tracks a count. It is not safe for concurrent use.
type Counter struct {
	value int
}

// Increment increases the count by one.
func (c *Counter) Increment() {
	c.value++
}
```

Do not create `service_helpers.go` or `service_utils.go` to scatter methods of the
same type. When responsibilities grow, extract a cohesive type instead.

## Formatting

* Run `gofmt` on changed Go files and use its formatting in documentation examples.
* Group standard-library imports separately from third-party imports.
* Use blank lines to separate logical steps, not every statement.
* Always insert one blank line after a complete `if`, `for`, or `switch` statement
  before the next statement in the same block. This includes `for range` loops,
  type switches, and control-flow blocks that end with an early return.
* Treat an `if` / `else if` / `else` chain as one statement: keep `} else {` together
  and put the blank line after the complete chain. No trailing blank line is needed
  when only the enclosing block's closing brace follows.

Add this spacing explicitly; `gofmt` does not insert the required blank lines.

## Naming

* Use descriptive, intention-revealing names.
* Avoid unnecessary abbreviations.
* Use verbs for functions (Build, Parse, Fetch).
* Use nouns for types (Parser, Client, Store).
* Use familiar short names such as `i`, `err`, `ctx`, and `ok` in narrow scopes.
* Preserve Go initialisms, such as `HTTPPath`, `TLSDir`, and `ID`.
* Do not rename public symbols solely to improve style or remove stutter.

Avoid stutter:

* Preferred: `type Client struct{}`
* Avoid: `type MyProjectClient struct{}`

## Variable Declarations

### Prefer var over := (with narrow exceptions)

Prefer `var` for zero values, explicit types, and related values reused across
branches. For example, this declaration fragment:

```go
var (
	count int
	ids   []string
	err   error
)
```

Group related declarations near their first use; do not hoist every local to the
beginning of a function.

Use `:=` only when it meaningfully improves clarity in tight scopes and the
declaration is simple. Loop variables and scoped lookups fit this exception too.

`:=` is acceptable when all of the following are true:

* The declaration is local and close to first use.
* The right-hand side is short and obvious.
* The variable has no ambiguity in meaning or type.
* The statement does not risk shadowing an existing variable.

Simple declaration fragments:

```go
done := make(chan struct{}, 1)
timeout := 5 * time.Second
```

When a value needs to survive a branch, declare it outside and assign with `=`.
For example, inside a function returning `([]byte, error)`:

```go
var (
	data []byte
	err  error
)

if data, err = os.ReadFile(filename); err != nil {
	return nil, fmt.Errorf("read configuration %q: %w", filename, err)
}

return data, nil
```

Scoped lookup fragment:

```go
if v, ok := m[key]; ok {
	return v, nil
}
```

### Avoid Shadowing

Do not shadow an existing local, parameter, receiver, or named result. An inner
`:=` can introduce a new variable even if the name already exists in an outer scope.

Avoid:

```go
// Incorrect: the inner err hides the result, so the caller receives nil.
func update() (err error) {
	if err := save(); err != nil {
		log.Print(err)
	}

	return err
}
```

Assign to the existing result and handle the error immediately:

```go
func update() (err error) {
	if err = save(); err != nil {
		return fmt.Errorf("save update: %w", err)
	}

	return nil
}
```

## Named Returns

### Named Returns Are Encouraged

Prefer named results when their names explain non-obvious outputs or distinguish
multiple values. They are encouraged for public APIs, but are not mandatory when
names add no information. Name all results in a signature or leave all unnamed.

### No Naked Returns

Never use a naked return in a function that returns values, including in early
exits. Explicit values make control flow easier to review and refactor. Bare
`return` is acceptable in any function or closure with no results.

Avoid:

```go
// Incorrect: the returned values are implicit.
func parseCount(input string) (count int, err error) {
	count, err = strconv.Atoi(input)
	return
}
```

Prefer:

```go
func parseCount(input string) (count int, err error) {
	if count, err = strconv.Atoi(input); err != nil {
		return 0, fmt.Errorf("parse count %q: %w", input, err)
	}

	return count, nil
}
```

Named results can be changed by deferred functions even after an explicit return.
Use that behavior deliberately for cleanup, keep it visible, and never shadow the
result being updated.

## Function Design

Keep functions focused on one responsibility. Fitting on one screen is a useful
heuristic, not a strict line-count rule. Extract helpers that name meaningful
operations, not merely to move lines. Extract a type when responsibilities diverge.

Use early returns to reduce nesting. For example, this statement fragment:

```go
if err != nil {
	return resp, err
}
```

## Plugin and Library Responsibilities

**Prefer plugins that load libraries and provide handlers for those libraries when
needed. This is a best practice, not a requirement.** Most request logic should
usually live in the library, with the plugin handling integration with Vroomy.

The preferred division is:

* The plugin configures and initializes the library through its lifecycle methods,
  exposes a shared backend when useful, and closes resources it owns.
* A plugin handler reads and decodes the inbound request, passes the relevant input
  to the library, then maps the library's result or error to the intended HTTP
  status, headers, and response body.
* The library handles business rules, domain validation, persistence, and the main
  processing for the request. Prefer APIs that accept ordinary Go values and return
  results and errors so this behavior can be tested independently of HTTP or Vroomy.

A plugin may only load a library or expose a backend; handlers are optional. The
library can be a package in the same module or a separate module. Use judgment for
small handlers and special cases; this guidance does not require extracting a
library for every trivial operation.

See [docs/plugins.md](docs/plugins.md#preferred-architecture) for the request flow
and how these responsibilities fit the plugin lifecycle.

## Error Handling

* Handle errors before using the associated result.
* For external operation errors (I/O, filesystem, network, system calls), add useful
  operation context and preserve the cause with `%w` when returning the error.
* Include a relevant identifier or path when appropriate. Avoid secrets and whole
  payloads, and do not repeat identical context at every layer.
* Return descriptive sentinel contract/state errors directly where the API expects
  them. Use `errors.Is` or `errors.As` for wrapped errors in new code, rather than
  comparing message text to identify a cause or type.
* Preserve existing error contracts. Changing `%v` to `%w` exposes a cause to callers
  and can change behavior; do not treat that as incidental formatting cleanup.

Contextual error fragment:

```go
if err != nil {
	return out, fmt.Errorf("load user %q: %w", id, err)
}
```

Return errors from library code. Prefer logging once at the boundary that handles
the failure; do not both log and return the same error without a specific reason.
Reserve `log.Fatal` and `os.Exit` for executable entry points, since they bypass
normal caller cleanup. Do not use panics for ordinary operational failures.

Sentinel check fragment:

```go
if closed {
	return ErrIsClosed
}
```

Use the actual sentinel promised by the API. Some Vroomy error constants are legacy
declarations that are not returned by current code.

### Resource Ownership

The code that acquires a resource owns cleanup unless it explicitly transfers
ownership. Arrange cleanup after successful acquisition, including when later
initialization fails. Check meaningful `Close`, flush, and final write errors.
If both the operation and cleanup fail, preserve the original failure and handle
the cleanup failure too. Explain an intentional ignored cleanup error briefly.

For example, this function preserves write and close errors using the
standard-library `errors` package:

```go
func writeState(filename string, data []byte) (err error) {
	var file *os.File
	if file, err = os.Create(filename); err != nil {
		return fmt.Errorf("create state %q: %w", filename, err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close state %q: %w", filename, closeErr))
		}
	}()

	if _, err = file.Write(data); err != nil {
		return fmt.Errorf("write state %q: %w", filename, err)
	}

	return nil
}
```

This demonstrates error handling, not atomic file replacement. If a file already
imports `github.com/gdbu/errors`, distinguish it from the standard library with a
clear alias when both are needed.

## Interfaces

* Prefer accepting small interfaces when they express a real dependency boundary.
  Do not create an interface for every concrete type.
* Prefer returning concrete types from constructors. Preserve required contracts
  such as `Plugin.Backend() interface{}` and factories returning
  `(httpserve.Handler, error)`.
* Keep interfaces small and behavior-focused.
* Define interfaces near where they are used.

## Receivers

* Use pointer receivers when mutating structs, when copying would be expensive,
  or when a struct contains synchronization primitives.
* Keep receiver choices consistent for a type and compatible with its interfaces.
  Do not copy a value containing a mutex after it has been used.
* Keep receiver names short (s, c, p).
* Never use `this` or `self` as receiver names.

## Comments

* All exported types, functions, methods, constants, and variables must have Go doc
  comments. Document public fields and interface methods when their contract needs
  explanation.
* Go doc comments must begin with the name of the exported symbol.
* API comments describe behavior and contracts: defaults, errors, side effects,
  ownership, and concurrency guarantees where relevant. Implementation comments
  explain non-obvious decisions and constraints.
* Keep comments concise and avoid restating obvious code, but do not omit necessary
  behavior to meet a line limit.
* Comment private helpers when the comment adds information or aids navigation.
* Keep documentation examples consistent with the guide. Label incorrect examples
  and code fragments explicitly. Do not promise that an unimplemented feature works.

The `Counter` example above documents behavior and keeps the constructor above the
type. Update the relevant guide when that behavior changes.

Avoid this redundant comment fragment:

```go
// i increments by 1.
i++
```

## Tests

* Use the standard `testing` package. Prefer table-driven tests for related cases;
  a single focused test does not need a table.
* Keep setup explicit.
* Avoid clever test abstractions.
* Use descriptive case names and `t.Run` where separate case output helps.
* Use `t.Helper`, `t.TempDir`, and `t.Cleanup` where appropriate for diagnostics,
  temporary files, and test-owned resources.
* Assert observable behavior and relevant error cases. Add focused regression
  coverage for bug fixes, not tests that merely mirror trivial declarations.
* Do not use `t.Parallel` when mutating Vroomy's global registry or process working
  directory. See [docs/development.md](docs/development.md) for isolation guidance.

For example, a test in package `vroomy` with an import of `testing`:

```go
func TestEnvironment_Must(t *testing.T) {
	type testcase struct {
		name    string
		env     Environment
		key     string
		want    string
		wantErr bool
	}

	tests := []testcase{
		{name: "present", env: Environment{"key": "value"}, key: "key", want: "value"},
		{name: "empty but present", env: Environment{"key": ""}, key: "key"},
		{name: "missing", env: Environment{}, key: "key", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got string
				err error
			)
			got, err = tt.env.Must(tt.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Must(%q) error = %v, want error %t", tt.key, err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("Must(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}
```

Follow nearby test naming. The repository already has a test with this name; adapt
existing coverage rather than duplicating it.

## Pull Request Descriptions

Use [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md) for every
pull request description. Preserve its Summary, Changes, and Testing sections and
replace the placeholders with details relevant to the change. In Testing, report
the checks actually run and any limitations.

## PR Checklist

Apply this checklist to the scope of the change:

* [ ] Pull request description follows [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md)
* [ ] `gofmt` applied
* [ ] Completed `if`, `for`, and `switch` statements are separated from following statements by a blank line
* [ ] One file per primary type
* [ ] Production methods stay with their type, except for documented build/generated cases
* [ ] Constructors appear directly above their type
* [ ] Functions are small and focused
* [ ] Named returns used appropriately
* [ ] No naked returns
* [ ] Declarations use `var` or a documented `:=` exception
* [ ] No shadowing
* [ ] External/operation errors include context
* [ ] Sentinel/state errors follow the API's contract
* [ ] Existing error contracts and resource ownership are preserved
* [ ] Exported symbols have Go doc comments
* [ ] Tests cover meaningful behavior changes and isolate shared state
* [ ] Relevant checks from [docs/development.md](docs/development.md) passed, or limitations are recorded
* [ ] Unrelated style changes are excluded and pre-existing user edits are preserved
