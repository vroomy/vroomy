package service

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
)

func newPanicLog() (pp *panicLog, err error) {
	var p panicLog
	if p.f, err = os.OpenFile("./panics.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0744); err != nil {
		return
	}

	pp = &p
	return
}

type panicLog struct {
	mu sync.Mutex
	f  *os.File
}

func (p *panicLog) Write(v interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	str := fmt.Sprintf("%v\n%s\n\n", v, string(debug.Stack()))

	if _, err := p.f.WriteString(str); err != nil {
		log.Println("Error writing string to panic log", err)
		return
	}

	if err := p.f.Sync(); err != nil {
		log.Println("Error writing string to panic log", err)
		return
	}
}

func (p *panicLog) Close() (err error) {
	return p.f.Close()
}
