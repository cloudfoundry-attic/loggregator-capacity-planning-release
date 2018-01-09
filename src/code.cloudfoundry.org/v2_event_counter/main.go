package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/datadogreporter"
	envstruct "code.cloudfoundry.org/go-envstruct"
	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
)

type Config struct {
	EventTitle    string `env:"EVENT_TITLE"`
	DatadogAPIKey string `env:"DATADOG_API_KEY, required"`
	JobName       string `env:"JOB_NAME,        required"`
	InstanceID    string `env:"INSTANCE_ID,     required"`
	Host          string `env:"HOST,            required"`

	CAPath   string `env:"CA_PATH,   required"`
	KeyPath  string `env:"KEY_PATH,  required"`
	CertPath string `env:"CERT_PATH, required"`

	LogProxyAddr string `env:"LOG_PROXY_ADDR, required"`
}

func main() {
	var cfg Config
	err := envstruct.Load(&cfg)
	if err != nil {
		log.Fatalf("failed to load config %s", err)
	}

	tlsConfig, err := loggregator.NewEgressTLSConfig(
		cfg.CAPath,
		cfg.CertPath,
		cfg.KeyPath,
	)
	if err != nil {
		log.Fatalf("failed to create tls config %s", err)
	}

	reader := newReader(cfg.EventTitle, cfg.LogProxyAddr, tlsConfig)
	go reader.run()

	reporter := datadogreporter.New(
		cfg.DatadogAPIKey,
		cfg.JobName,
		cfg.InstanceID,
		reader,
		datadogreporter.WithHost(cfg.Host),
	)

	reporter.Run()
}

type reader struct {
	title      string
	client     *loggregator.EnvelopeStreamConnector
	eventCount int64
}

func newReader(title string, logProxyAddr string, tlsConfig *tls.Config) *reader {
	return &reader{
		title: title,
		client: loggregator.NewEnvelopeStreamConnector(logProxyAddr, tlsConfig,
			loggregator.WithEnvelopeStreamLogger(log.New(os.Stdout, "", log.LstdFlags)),
		),
	}
}

func (r *reader) run() {
	stream := r.client.Stream(context.Background(), &loggregator_v2.EgressBatchRequest{
		ShardId:          fmt.Sprintf("%d", time.Now().UnixNano()),
		UsePreferredTags: true,
		Selectors: []*loggregator_v2.Selector{
			{
				Message: &loggregator_v2.Selector_Event{
					Event: &loggregator_v2.EventSelector{},
				},
			},
		},
	})

	for {
		envelopes := stream()
		log.Printf("Read %d envelopes", len(envelopes))
		for _, env := range envelopes {
			if env.GetEvent().GetTitle() == r.title {
				atomic.AddInt64(&r.eventCount, 1)
			}
		}
	}
}

func (r *reader) BuildPoints() []datadogreporter.Point {
	count := atomic.SwapInt64(&r.eventCount, 0)

	return []datadogreporter.Point{
		{
			Metric: "v2_event_counter.read",
			Points: [][]int64{
				[]int64{time.Now().Unix(), count},
			},
			Type: "guage",
		},
	}
}
