package emitter

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/v1"
)

type Client interface {
	EmitCounter(name string, opts ...loggregator.EmitCounterOption)
}

type Emitter struct {
	client           Client
	metricsPerSecond uint
	sentCount        int64
}

func New(
	caPath string,
	certPath string,
	keyPath string,
	apiVersion string,
	origin string,
	metricsPerSecond uint,
) *Emitter {
	var client Client
	var err error
	switch apiVersion {
	case "v1":
		client, err = v1.NewClient(
			v1.WithStringTag("origin", origin),
		)
		if err != nil {
			log.Fatalf("failed to create v1 client: %s", err)
		}
	case "v2":
		tlsConf, err := loggregator.NewIngressTLSConfig(caPath, certPath, keyPath)
		if err != nil {
			log.Fatalf("failed to create v2 tls config: %s", err)
		}

		client, err = loggregator.NewIngressClient(tlsConf,
			loggregator.WithStringTag("origin", origin),
		)
		if err != nil {
			log.Fatalf("failed to create v2 client: %s", err)
		}
	default:
		log.Fatalf("Invalid api-version, must be 'v1' or 'v2'")
	}

	return &Emitter{
		client:           client,
		metricsPerSecond: metricsPerSecond,
	}
}

func (e *Emitter) Run() {
	ns := time.Second / time.Duration(e.metricsPerSecond)

	var metricNames []string
	for i := 0; i < 100; i++ {
		metricNames = append(metricNames, fmt.Sprintf("capacity-planning-metric-%d", i))
	}

	ticker := time.NewTicker(ns)
	for range ticker.C {
		idx := atomic.LoadInt64(&e.sentCount) % int64(len(metricNames))
		e.client.EmitCounter(metricNames[idx])
		atomic.AddInt64(&e.sentCount, 1)
	}
}

func (e *Emitter) SwapCount() int64 {
	return atomic.SwapInt64(&e.sentCount, 0)
}