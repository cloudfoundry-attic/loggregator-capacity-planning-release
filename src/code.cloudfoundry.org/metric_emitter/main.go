package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/datadogreporter"
	"code.cloudfoundry.org/metric_emitter/internal/emitter"
)

var (
	tlsConfig *tls.Config = &tls.Config{
		InsecureSkipVerify: true,
	}

	httpClient *http.Client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
)

func main() {
	apiVersion := flag.String("api-version", "", "Version of API to write metrics to. (v1 or v2)")
	origin := flag.String("origin", "", "Origin to be applied to all outgoing envelopes")
	metricsPerSecond := flag.Uint("metrics-per-second", 1000, "Number of counter events to be emitted per second")
	datadogAPIKey := flag.String("datadog-api-key", "", "Datadog API key.")
	jobName := flag.String("job-name", "", "Name of the bosh job")
	instanceID := flag.String("instance-id", "", "Bosh job instance ID")

	caPath := flag.String("ca-path", "", "Path to certificate authority cert")
	certPath := flag.String("cert-path", "", "Path to client certificate for connecting to metron")
	keyPath := flag.String("key-path", "", "Path to private key for connecting to metron")
	flag.Parse()

	var missing []string
	if *origin == "" {
		missing = append(missing, "origin")
	}

	if *datadogAPIKey == "" {
		missing = append(missing, "datadog-api-key")
	}

	if *jobName == "" {
		missing = append(missing, "job-name")
	}

	if *instanceID == "" {
		missing = append(missing, "instance-id")
	}

	if len(missing) > 0 {
		log.Fatalf("missing required flags: %s", strings.Join(missing, ", "))
	}

	emitter := emitter.New(
		*caPath,
		*certPath,
		*keyPath,
		*apiVersion,
		*origin,
		*metricsPerSecond,
	)
	go emitter.Run()

	reporter := datadogreporter.New(
		*datadogAPIKey,
		*jobName,
		*instanceID,
		emitter,
	)
	reporter.Run()
}
