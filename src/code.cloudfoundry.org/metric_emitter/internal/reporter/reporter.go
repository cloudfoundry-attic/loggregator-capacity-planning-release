package reporter

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"code.cloudfoundry.org/metric_emitter/internal/emitter"
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
	emitter       *emitter.Emitter
	httpClient    *http.Client
}

func New(
	datadogAPIKey string,
	jobName string,
	instanceID string,
	emitter *emitter.Emitter,
	httpClient *http.Client,
) *Reporter {
	return &Reporter{
		datadogAPIKey: datadogAPIKey,
		jobName:       jobName,
		instanceID:    instanceID,
		emitter:       emitter,
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
		sent := r.emitter.SwapCount()

		data, err := r.buildMessageBody(sent)
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

func (r *Reporter) buildMessageBody(sent int64) ([]byte, error) {
	currentTime := time.Now().Unix()

	points := []Point{
		{
			Metric: "capacity_planning.sent",
			Points: [][]int64{
				[]int64{currentTime, sent},
			},
			Type: "gauge",
			Tags: []string{
				"event_type:metrics",
				"job_name:" + r.jobName,
				"instance_index:" + r.instanceID,
			},
		},
	}

	body := map[string][]Point{"series": points}

	return json.Marshal(&body)
}
