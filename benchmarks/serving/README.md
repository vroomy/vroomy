# Serving allocations

`serving_benchmark_test.go` measures warmed request handling using the pinned apex
router, Vroomy's configured handler resolver, and apex's middleware wrapper. The
engine does not expose `ServeHTTP`, so the harness registers the resolved chain on
a public `apex.Router`. Requests and a response writer that discards bodies are
created before timing. Startup, network I/O, and response recording are excluded.
The writer reuses its header map; `Header.Set` still allocates replacement values.

`BenchmarkPluginHTTPServing` also measures a real HTTP/1.1 keep-alive connection
to the same router. Its client writes a prebuilt request and reads headers into a
reused buffer. Transport costs are included, with no per-request `http.Client`
or response recorder allocations. This benchmark needs local socket permissions.

Run from the repository root:

```sh
go test -run '^$' -bench 'Benchmark(Plugin|Redirect)Serving' -benchmem -benchtime=200ms -count=5 .
go test -run '^$' -bench '^BenchmarkPluginHTTPServing$' -benchmem -benchtime=200ms -count=5 .
go test -o /tmp/vroomy-serving.test -run '^$' -bench '^BenchmarkPluginServing/(Direct|String|MiddlewareParam)$' -benchtime=100000x -memprofilerate=1 -memprofile=/tmp/vroomy-serving.mem .
go tool pprof -top -alloc_objects /tmp/vroomy-serving.test /tmp/vroomy-serving.mem
go test -o /tmp/vroomy-serving.test -run '^$' -bench '^BenchmarkPluginServing/(Direct|MiddlewareParam)$' -benchtime=1s -cpuprofile=/tmp/vroomy-serving.cpu .
go tool pprof -top -nodecount=20 /tmp/vroomy-serving.test /tmp/vroomy-serving.cpu
go test -o /tmp/vroomy-http.test -run '^$' -bench '^BenchmarkPluginHTTPServing$' -benchtime=10000x -memprofilerate=1 -memprofile=/tmp/vroomy-http.mem .
go tool pprof -top -alloc_objects /tmp/vroomy-http.test /tmp/vroomy-http.mem
```

Use a writable temporary `GOCACHE` if required by the sandbox. Run benchmarks and
profiles separately, without the race detector. Profiles include setup and test
harness costs; use `-benchmem` for per-request counts. Pool misses after garbage
collection can allocate, so warmed zero-allocation results do not promise that
every request allocates zero bytes.

`before.txt` records five baseline samples on Go 1.27.2, darwin/arm64, Apple M3,
using apex v0.16.1. `after.txt` uses the same commands and machine. HTTP samples
are in `http-before.txt` and `http-after.txt`; the original `utils.go` from
`b277177` was supplied with a Go build overlay to measure the unchanged HTTP
harness against the original dispatch. Binary profiles are temporary artifacts
and are not committed; text summaries are retained here.

## Results

Timings below are medians of five 200 ms samples, useful as local observations
rather than statistically established speedups. Allocation counts were consistent
across the samples. Startup and pool warmup are excluded.

| Request path | Allocs/op before → after | B/op before → after | Median ns/op before → after |
| --- | --- | --- | --- |
| Direct method, no content | 1 → 0 | 8 → 0 | 168.9 → 109.4 |
| Factory, no content | 0 → 0 | 0 → 0 | 46.42 → 45.07 |
| Direct method, bytes | 2 → 1 | 24 → 16 | 182.2 → 126.7 |
| Direct method, string | 3 → 2 | 32 → 24 | 193.9 → 139.8 |
| Middleware + parameter handler | 2 → 0 | 16 → 0 | 337.8 → 224.2 |
| HTTPS redirect GET | 4 → 3 | 100 → 96 | 98.96 → 81.96 |
| Redirect listener POST/404 | 2 → 1 | 32 → 16 | 34.14 → 25.98 |
| HTTP/1.1 keep-alive, no content | 14 → 13 | ~1.4 KiB → ~1.4 KiB | 19077 → 18955 |

## Review findings and remaining costs

- [Handler resolution](../../utils.go): converting a reflected bound method to a
  function left `reflect.methodReceiver` on every request, allocating 8 bytes per
  method invocation. Calling its unbound function with the captured receiver
  removes that allocation. Argument storage belongs to each invocation, keeping
  concurrent requests independent. Regression tests cover pointer, value, and
  promoted receivers, distinct plugin instances, and concurrent requests.
- [Redirect listener](../../redirectListener.go): the configured port string and
  fixed 404 body are now prepared once. Remaining GET allocations construct the
  host/port, serialize the destination URL, and set the Location header. The
  POST/404 allocation sets Content-Type. Tests preserve escaped paths, query
  strings, IPv6, the body, write-error reporting, and request URL ownership.
- The before/after allocation profiles attribute the remaining string-response
  allocations to apex `Context.WriteString` (conversion to bytes) and
  `net/textproto.MIMEHeader.Set` (header value storage). Reusable response bytes
  avoid the conversion; ordinary header writes still allocate in apex v0.16.1.
- `http-allocs-after.txt` attributes the remaining loopback costs to Go's request
  and MIME-header parsing, URL parsing, cancellation contexts, and background
  connection reads. Zero total HTTP allocations would require transport changes
  beyond Vroomy's handler dispatch.
- CPU profiles still show reflection in direct dispatch. Existing factories
  return compiled Go functions and remain the fastest measured option for
  handlers where CPU overhead matters; they already allocate zero for dispatch.
- Config parsing, dependency injection, group composition, and route registration
  run during startup. Legacy `Response` constructors are outside configured
  routing. Panic stack logging and error construction are failure-path costs.
  These were left unchanged.

JSON/form processing, TLS, live ACME, and application-specific handlers were not
performance-profiled. Functional HTTP/TLS tests are separate from these allocation
measurements; the results do not claim zero allocations for every application.

## Validation

`go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...` passed.
The allocation regression test fails against the original dispatch and passes
with the optimization, including under the race detector. Both documented hello
endpoints returned `PONG` and the example was stopped with Ctrl-C. The HTTP
benchmark also passed a separate race check; race-instrumented timings and
allocation counts are excluded from the performance comparison.
