package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/datadogreporter"
	"code.cloudfoundry.org/syslog_counter/internal/sysloglistener"
)

var (
	tlsConfig *tls.Config = &tls.Config{}

	httpClient *http.Client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	datadogAPIKey := flag.String("datadog-api-key", "", "Datadog API key.")

	jobName := flag.String("job-name", "", "Name of the bosh job")
	instanceID := flag.String("instance-id", "", "Bosh job instance ID")

	var missing []string
	if *port == "" {
		missing = append(missing, "port")
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

	lis := sysloglistener.New(*port)
	go lis.Run()

	reporter := datadogreporter.New(
		*datadogAPIKey,
		*jobName,
		*instanceID,
		lis,
		httpClient,
	)
	reporter.Run()
}
