package reporter

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"code.cloudfoundry.org/eventcounter/internal/reader"
)

const datadogAddr = "https://app.datadoghq.com/api/v1/series"

type Point struct {
	Metric string    `json:"metric"`
	Points [][]int64 `json:"points"`
	Type   string    `json:"type"`
	Host   string    `json:"host"`
	Tags   []string  `json:"tags"`
}

type Reporter struct {
	datadogAPIKey string
	jobName       string
	instanceID    string
	reader        *reader.Reader
	httpClient    *http.Client
}

func New(
	datadogAPIKey string,
	jobName string,
	instanceID string,
	reader *reader.Reader,
	httpClient *http.Client,
) *Reporter {
	return &Reporter{
		datadogAPIKey: datadogAPIKey,
		jobName:       jobName,
		instanceID:    instanceID,
		reader:        reader,
		httpClient:    httpClient,
	}
}

func (r *Reporter) Run() {
	dURL, err := url.Parse(datadogAddr)
	if err != nil {
		log.Fatalf("Failed to parse datadog URL: %s", err)
	}
	query := url.Values{
		"api_key": []string{r.datadogAPIKey},
	}
	dURL.RawQuery = query.Encode()

	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		logs, metrics := r.reader.SwapCounts()

		data, err := r.buildMessageBody(logs, metrics)
		if err != nil {
			log.Printf("failed to build request body for datadog: %s", err)
			continue
		}

		log.Printf("Sending data to datadog: %s", data)

		response, err := r.httpClient.Post(dURL.String(), "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("failed to post to datadog: %s", err)
			continue
		}

		if response.StatusCode > 299 || response.StatusCode < 200 {
			log.Printf("Expected successful status code from Datadog, got %d", response.StatusCode)
			continue
		}
	}
}

func (r *Reporter) buildMessageBody(logs, metrics int64) ([]byte, error) {
	currentTime := time.Now().Unix()

	points := []Point{
		{
			Metric: "capacity_planning.received",
			Points: [][]int64{
				[]int64{currentTime, logs},
			},
			Type: "gauge",
			Tags: []string{
				"logs",
				"job_name:" + r.jobName,
				"instance_index:" + r.instanceID,
			},
		},
		{
			Metric: "capacity_planning.received",
			Points: [][]int64{
				[]int64{currentTime, metrics},
			},
			Type: "gauge",
			Tags: []string{
				"metrics",
				"job_name:" + r.jobName,
				"instance_index:" + r.instanceID,
			},
		},
	}

	body := map[string][]Point{"series": points}

	return json.Marshal(&body)
}
