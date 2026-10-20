# Serving allocations

`serving_benchmark_test.go` measures warmed request handling using the pinned apex
router, Vroomy's configured handler resolver, and apex's middleware wrapper. The
engine does not expose `ServeHTTP`, so the harness registers the resolved chain on
a public `apex.Router`. Requests and a response writer that discards bodies are
created before timing. Startup, network I/O, and response recording are excluded.
The writer reuses its header map; `Header.Set` still allocates replacement values.

Run from the repository root:

```sh
go test -run '^$' -bench 'Benchmark(Plugin|Redirect)Serving' -benchmem -benchtime=200ms -count=5 .
go test -o /tmp/vroomy-serving.test -run '^$' -bench '^BenchmarkPluginServing/(Direct|String|MiddlewareParam)$' -benchtime=100000x -memprofilerate=1 -memprofile=/tmp/vroomy-serving.mem .
go tool pprof -top -alloc_objects /tmp/vroomy-serving.test /tmp/vroomy-serving.mem
go test -o /tmp/vroomy-serving.test -run '^$' -bench '^BenchmarkPluginServing/(Direct|MiddlewareParam)$' -benchtime=1s -cpuprofile=/tmp/vroomy-serving.cpu .
go tool pprof -top /tmp/vroomy-serving.test /tmp/vroomy-serving.cpu
```

Use a writable temporary `GOCACHE` if required by the sandbox. Run benchmarks and
profiles separately, without the race detector. Profiles include setup and test
harness costs; use `-benchmem` for per-request counts. Pool misses after garbage
collection can allocate, so warmed zero-allocation results do not promise that
every request allocates zero bytes.

`before.txt` records five baseline samples on Go 1.27.2, darwin/arm64, Apple M3,
using apex v0.16.1. The allocation profile identifies `reflect.methodReceiver`,
`net/textproto.MIMEHeader.Set`, and apex `Context.WriteString` as the request-path
allocation sites. Binary profiles are temporary artifacts and are not committed.
