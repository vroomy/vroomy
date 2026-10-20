package vroomy

import "testing"

func TestRouteString(t *testing.T) {
	var route = Route{HTTPPath: "/ping", Target: "public", Handlers: []string{"hello.Ping", "hello.Message(a)"}}
	const want = `{ HTTPPath: "/ping", Target: "public" Plugin Handler: "[hello.Ping hello.Message(a)]" }`
	if got := route.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
