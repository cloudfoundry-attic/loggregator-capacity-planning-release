package main

import (
	"flag"
	"log"
	"strings"

	"code.cloudfoundry.org/datadogreporter"
	"code.cloudfoundry.org/syslog_counter/internal/sysloglistener"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	datadogAPIKey := flag.String("datadog-api-key", "", "Datadog API key.")

	jobName := flag.String("job-name", "", "Name of the bosh job")
	instanceID := flag.String("instance-id", "", "Bosh job instance ID")

	flag.Parse()

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
	)
	reporter.Run()
}
