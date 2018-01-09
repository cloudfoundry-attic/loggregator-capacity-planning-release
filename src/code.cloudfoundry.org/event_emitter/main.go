package main

import (
	"context"
	"crypto/tls"
	"log"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/datadogreporter"
	envstruct "code.cloudfoundry.org/go-envstruct"
	loggregator "code.cloudfoundry.org/go-loggregator"
)

type Config struct {
	EmitInterval  time.Duration `env:"EMIT_INTERVAL"`
	EventTitle    string        `env:"EVENT_TITLE"`
	EventBody     string        `env:"EVENT_BODY"`
	DatadogAPIKey string        `env:"DATADOG_API_KEY, required"`
	JobName       string        `env:"JOB_NAME,        required"`
	InstanceID    string        `env:"INSTANCE_ID,     required"`
	Host          string        `env:"HOST,            required"`

	CAPath   string `env:"CA_PATH,   required"`
	KeyPath  string `env:"KEY_PATH,  required"`
	CertPath string `env:"CERT_PATH, required"`
}

func main() {
	cfg := Config{
		EmitInterval: time.Second,
	}
	err := envstruct.Load(&cfg)
	if err != nil {
		log.Fatalf("failed to load config from environment: %s", err)
	}

	tlsConfig, err := loggregator.NewIngressTLSConfig(
		cfg.CAPath,
		cfg.CertPath,
		cfg.KeyPath,
	)
	if err != nil {
		log.Fatalf("failed to build TLS config: %s", err)
	}

	wr := newWriter(cfg.EmitInterval, cfg.EventTitle, cfg.EventBody, tlsConfig)
	go wr.run()

	reporter := datadogreporter.New(
		cfg.DatadogAPIKey,
		cfg.JobName,
		cfg.InstanceID,
		wr,
		datadogreporter.WithHost(cfg.Host),
	)
	reporter.Run()
}

type writer struct {
	emitInterval time.Duration
	client       *loggregator.IngressClient
	eventCount   int64
	title        string
	body         string
}

func newWriter(
	emitInterval time.Duration,
	title string,
	body string,
	tlsConfig *tls.Config,
) *writer {
	c, err := loggregator.NewIngressClient(tlsConfig)
	if err != nil {
		log.Fatalf("failed to create ingress client: %s", err)
	}

	return &writer{
		emitInterval: emitInterval,
		client:       c,
		title:        title,
		body:         body,
	}
}

func (w *writer) run() {
	t := time.NewTicker(w.emitInterval)
	for range t.C {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := w.client.EmitEvent(ctx, w.title, w.body)
		cancel()
		if err != nil {
			log.Printf("failed to write event: %s", err)
			continue
		}

		atomic.AddInt64(&w.eventCount, 1)
	}
}

func (w *writer) BuildPoints() []datadogreporter.Point {
	count := atomic.SwapInt64(&w.eventCount, 0)

	return []datadogreporter.Point{
		{
			Metric: "event_emitter.sent",
			Points: [][]int64{
				[]int64{time.Now().Unix(), count},
			},
			Type: "gauge",
		},
	}
}
