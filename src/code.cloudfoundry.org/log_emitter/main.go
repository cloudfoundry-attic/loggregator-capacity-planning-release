package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/authenticator"
	"code.cloudfoundry.org/datadogreporter"
	"code.cloudfoundry.org/log_emitter/internal/reader"
	"code.cloudfoundry.org/log_emitter/internal/writer"
)

var (
	logMessage       string
	messagesSent     int64
	messagesReceived int64

	tlsConfig *tls.Config = &tls.Config{
		InsecureSkipVerify: true,
	}

	httpClient *http.Client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	reportReadMessages bool
)

const datadogAddr = "https://app.datadoghq.com/api/v1/series"

type VCAPApplication struct {
	APIAddr string `json:"cf_api"`
	AppID   string `json:"application_id"`
	AppName string `json:"application_name"`
}

type V2Info struct {
	DopplerAddr string `json:"doppler_logging_endpoint"`
	UAAAddr     string `json:"token_endpoint"`
}

type AuthInfo struct {
	ClientID     string
	ClientSecret string
}

func main() {
	logsPerSecond := flag.Uint("logs-per-second", 1000, "Log messages to emit per second. Default: 1000")
	logSize := flag.Uint("log-bytes", 1000, "Length of log messages in bytes. Default: 1000")
	datadogAPIKey := flag.String("datadog-api-key", "", "Datadog API key for emitting metrics.")

	var authInfo AuthInfo
	flag.StringVar(&authInfo.ClientID, "client-id", "", "ID of client used for authentication.")
	flag.StringVar(&authInfo.ClientSecret, "client-secret", "", "Secret used for authentication.")

	flag.Parse()
	vcapApp := loadVCAP()

	for i := uint(0); i < *logSize; i++ {
		logMessage += "?"
	}

	var r *reader.Reader
	instanceID := os.Getenv("INSTANCE_INDEX")
	reportReadMessages = instanceID == "0"
	if reportReadMessages {
		v2Info, err := getV2Info(vcapApp.APIAddr)
		if err != nil {
			log.Fatalf("failed to get API info: %s", err)
		}

		auth := authenticator.New(
			authInfo.ClientID,
			authInfo.ClientSecret,
			v2Info.UAAAddr,
		)

		r = reader.New(
			v2Info.DopplerAddr,
			vcapApp.AppID,
			logMessage,
			auth,
		)

		go r.Run()
	}

	w := writer.New(logMessage, *logsPerSecond)
	go w.Run()

	reporter := datadogreporter.New(
		*datadogAPIKey,
		vcapApp.AppName,
		instanceID,
		NewReportWrapper(vcapApp.AppName, r, w),
		datadogreporter.WithHost(vcapApp.APIAddr),
	)
	reporter.Run()
}

type ReporterWrapper struct {
	appName string
	reader  *reader.Reader
	writer  *writer.Writer
}

func NewReportWrapper(appName string, r *reader.Reader, w *writer.Writer) *ReporterWrapper {
	return &ReporterWrapper{appName: appName, reader: r, writer: w}
}

func (rw *ReporterWrapper) BuildPoints() []datadogreporter.Point {
	currentTime := time.Now().Unix()

	writeCount := rw.writer.Count()
	points := []datadogreporter.Point{
		{
			Metric: "capacity_planning.sent",
			Points: [][]int64{{currentTime, writeCount}},
			Type:   "gauge",
			Tags:   []string{rw.appName, "event_type:logs"},
		},
	}

	if rw.reader != nil {
		readCount := rw.reader.Count()
		points = append(points, datadogreporter.Point{
			Metric: "capacity_planning.received",
			Points: [][]int64{{currentTime, readCount}},
			Type:   "gauge",
			Tags:   []string{rw.appName, "event_type:logs"},
		})
	}

	return points
}

func loadVCAP() *VCAPApplication {
	var vcapApp VCAPApplication
	err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &vcapApp)
	if err != nil {
		log.Fatalf("failed to unmarshal VCAP_APPLICATION: %s", err)
	}

	return &vcapApp
}

func getV2Info(uaaAddr string) (*V2Info, error) {
	response, err := httpClient.Get(uaaAddr + "/v2/info")
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected 200 status code from /v2/info, got %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}

	var v2Info V2Info
	err = json.Unmarshal(body, &v2Info)
	if err != nil {
		return nil, err
	}

	return &v2Info, nil
}
