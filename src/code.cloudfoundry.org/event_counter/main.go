package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/datadogreporter"
	"code.cloudfoundry.org/event_counter/internal/authenticator"
	"code.cloudfoundry.org/event_counter/internal/reader"
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
	loggregatorEgressURL := flag.String("loggregator-egress-url", "", "Websocket URL for Loggregator egress.")
	datadogAPIKey := flag.String("datadog-api-key", "", "Datadog API key.")
	subscriptionID := flag.String("subscription-id", "capacity-planning", "The firehose subscription ID")
	counterOrigin := flag.String("counter-origin", "", "Count only metrics from exactly this origin.")

	jobName := flag.String("job-name", "", "Name of the bosh job")
	instanceID := flag.String("instance-id", "", "Bosh job instance ID")

	uaaAddr := flag.String("uaa-addr", "", "The URL for UAA")
	clientID := flag.String("client-id", "", "ID of client used for authentication.")
	clientSecret := flag.String("client-secret", "", "Secret used for authentication.")
	flag.Parse()

	var missing []string
	if *loggregatorEgressURL == "" {
		missing = append(missing, "loggregator-egress-url")
	}

	if *datadogAPIKey == "" {
		missing = append(missing, "datadog-api-key")
	}

	if *subscriptionID == "" {
		missing = append(missing, "subscription-id")
	}

	if *counterOrigin == "" {
		missing = append(missing, "counter-origin")
	}

	if *jobName == "" {
		missing = append(missing, "job-name")
	}

	if *instanceID == "" {
		missing = append(missing, "instance-id")
	}

	if *uaaAddr == "" {
		missing = append(missing, "uaa-addr")
	}

	if *clientID == "" {
		missing = append(missing, "client-id")
	}

	if *clientSecret == "" {
		missing = append(missing, "client-secret")
	}

	if len(missing) > 0 {
		log.Fatalf("missing required flags: %s", strings.Join(missing, ", "))
	}

	reader := reader.New(
		authenticator.New(*clientID, *clientSecret, *uaaAddr, httpClient),
		*loggregatorEgressURL,
		*subscriptionID,
		*counterOrigin,
		tlsConfig,
	)
	reporter := datadogreporter.New(
		*datadogAPIKey,
		*jobName,
		*instanceID,
		reader,
		httpClient,
	)

	go reader.Run()

	reporter.Run()
}
