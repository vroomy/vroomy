package main

import (
	"fmt"
	"github.com/hatchify/scribe"
	"runtime/debug"
)

func newPanicLog() (pp *panicLog, err error) {
	var p panicLog
	stdout := scribe.NewStdout()
	p.f = scribe.NewWithWriter(stdout, ":: panic ::")
	pp = &p
	return
}

type panicLog struct {
	f  *scribe.Scribe
}

func (p *panicLog) Write(v interface{}) {
	str := fmt.Sprintf("%v\n%s\n\n", v, string(debug.Stack()))

	p.f.Error(str)
}
