package writer

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Writer struct {
	logMsg        string
	sentMsgs      int64
	logsPerSecond uint
}

func New(logMsg string, logsPerSecond uint) *Writer {
	return &Writer{
		logMsg:        logMsg,
		logsPerSecond: logsPerSecond,
	}
}

func (w *Writer) Count() int64 {
	return atomic.SwapInt64(&w.sentMsgs, 0)
}

func (w *Writer) Run() {
	interval := time.Second / time.Duration(w.logsPerSecond)
	for {
		startTime := time.Now()
		w.emitLog()
		timeToSleep := interval - time.Since(startTime)
		time.Sleep(timeToSleep)
	}
}

func (w *Writer) emitLog() {
	atomic.AddInt64(&w.sentMsgs, 1)
	fmt.Printf("%s\n", w.logMsg)
}
