package datadogreporter

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

const datadogAddr = "https://app.datadoghq.com/api/v1/series"

type DatadogReporter struct {
	apiKey       string
	jobName      string
	instanceID   string
	pointBuilder pointBuilder
	httpClient   httpClient
	interval     time.Duration
}

func New(
	apiKey string,
	jobName string,
	instanceID string,
	pointBuilder pointBuilder,
	httpClient httpClient,
	opts ...reporterOpt,
) *DatadogReporter {
	r := &DatadogReporter{
		apiKey:       apiKey,
		jobName:      jobName,
		instanceID:   instanceID,
		pointBuilder: pointBuilder,
		httpClient:   httpClient,
		interval:     time.Minute,
	}

	for _, o := range opts {
		o(r)
	}
	return r
}

func (r *DatadogReporter) Run() {
	dURL, err := url.Parse(datadogAddr)
	if err != nil {
		log.Fatalf("Failed to parse datadog URL: %s", err)
	}
	query := url.Values{
		"api_key": []string{r.apiKey},
	}
	dURL.RawQuery = query.Encode()

	ticker := time.NewTicker(r.interval)
	for range ticker.C {
		body, err := r.buildRequestBody()
		if err != nil {
			log.Printf("failed to build request body for datadog: %s", err)
			continue
		}

		log.Printf("Sending point to datadog: %s", body)

		response, err := r.httpClient.Post(dURL.String(), "application/json", body)
		if err != nil {
			log.Printf("failed to post to datadog: %s", err)
			continue
		}

		respBody, _ := ioutil.ReadAll(response.Body)
		response.Body.Close()

		if response.StatusCode > 299 || response.StatusCode < 200 {
			log.Printf("Expected successful status code from Datadog, got %d", response.StatusCode)
			log.Printf("Response: %s", respBody)
			continue
		}
	}
}

func (r *DatadogReporter) buildRequestBody() (io.Reader, error) {
	points := r.pointBuilder.BuildPoints()

	for i, p := range points {
		p.Tags = append(p.Tags, "job_name:"+r.jobName)
		p.Tags = append(p.Tags, "instance_index:"+r.instanceID)

		points[i] = p
	}

	data, err := json.Marshal(map[string][]Point{"series": points})
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(data), nil
}

type Point struct {
	Metric string    `json:"metric"`
	Points [][]int64 `json:"points"`
	Type   string    `json:"type"`
	Host   string    `json:"host"`
	Tags   []string  `json:"tags"`
}

type pointBuilder interface {
	BuildPoints() []Point
}

type httpClient interface {
	Post(string, string, io.Reader) (*http.Response, error)
}

type reporterOpt func(*DatadogReporter)

func WithInterval(d time.Duration) reporterOpt {
	return func(r *DatadogReporter) {
		r.interval = d
	}
}
