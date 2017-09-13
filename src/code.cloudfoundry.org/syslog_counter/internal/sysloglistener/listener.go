package sysloglistener

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/datadogreporter"
	"code.cloudfoundry.org/rfc5424"
)

type SyslogListener struct {
	logCount int64
	port     string
}

func New(port string) *SyslogListener {
	return &SyslogListener{port: port}
}

func (sl *SyslogListener) Run() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%s", sl.port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	log.Printf("Listening on %s", sl.port)

	for {
		conn, err := l.Accept()
		log.Printf("Accepted connection")
		if err != nil {
			log.Printf("Error accepting: %s", err)
			continue
		}

		go sl.handle(conn)
	}
}

func (sl *SyslogListener) handle(conn net.Conn) {
	defer conn.Close()

	var msg rfc5424.Message
	for {
		_, err := msg.ReadFrom(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("ReadFrom err: %s", err)
			return
		}

		atomic.AddInt64(&sl.logCount, 1)
	}
}

func (sl *SyslogListener) BuildPoints() []datadogreporter.Point {
	count := atomic.SwapInt64(&sl.logCount, 0)

	return []datadogreporter.Point{
		{
			Metric: "capacity_planning.syslog-drain-received",
			Points: [][]int64{
				[]int64{time.Now().Unix(), count},
			},
			Type: "gauge",
			Tags: []string{
				"event_type:logs",
			},
		},
	}
}
