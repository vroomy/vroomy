package service

import (
	"fmt"
	"os"
	"sync"
)

func newPanicLog() (pp *panicLog, err error) {
	var p panicLog
	if p.f, err = os.OpenFile("./panics.log", os.O_CREATE|os.O_APPEND, 0744); err != nil {
		return
	}

	pp = &p
	return
}

type panicLog struct {
	mu sync.Mutex
	f  *os.File
}

func (p *panicLog) Write(v interface{}) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	str := fmt.Sprint(v)
	if _, err = p.f.WriteString(str); err != nil {
		return
	}

	return p.f.Sync()
}

func (p *panicLog) Close() (err error) {
	return p.f.Close()
}
