// Package plugins registers the hello application's plugins when imported.
package plugins

import (
	"github.com/vroomy/httpserve"
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
func (p *helloPlugin) Ping(ctx *httpserve.Context) {
	ctx.WriteString(200, "text/plain", "PONG\n")
}
