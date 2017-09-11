package reader

import (
	"crypto/tls"
	"log"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/datadogreporter"
	"code.cloudfoundry.org/event_counter/internal/authenticator"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"
)

type Reader struct {
	a              *authenticator.Authenticator
	egressAddr     string
	subscriptionID string
	counterOrigin  string
	tlsConfig      *tls.Config
	logCount       int64
	metricCount    int64
}

func New(
	a *authenticator.Authenticator,
	egressAddr string,
	subscriptionID string,
	counterOrigin string,
	tlsConfig *tls.Config,
) *Reader {
	return &Reader{
		a:              a,
		egressAddr:     egressAddr,
		subscriptionID: subscriptionID,
		counterOrigin:  counterOrigin,
		tlsConfig:      tlsConfig,
	}
}

func (r *Reader) Run() {
	for {
		authToken, err := r.a.Token()
		if err != nil {
			log.Printf("failed to authenticate with UAA: %s", err)
			time.Sleep(time.Second)
			continue
		}

		r.read(authToken)
	}
}

func (r *Reader) BuildPoints() []datadogreporter.Point {
	logs := atomic.SwapInt64(&r.logCount, 0)
	metrics := atomic.SwapInt64(&r.metricCount, 0)

	currentTime := time.Now().Unix()

	return []datadogreporter.Point{
		{
			Metric: "capacity_planning.received",
			Points: [][]int64{
				[]int64{currentTime, logs},
			},
			Type: "gauge",
			Tags: []string{
				"event_type:logs",
			},
		},
		{
			Metric: "capacity_planning.received",
			Points: [][]int64{
				[]int64{currentTime, metrics},
			},
			Type: "gauge",
			Tags: []string{
				"event_type:metrics",
			},
		},
	}
}

func (r *Reader) read(authToken string) {
	cmr := consumer.New(r.egressAddr, r.tlsConfig, nil)

	msgChan, errChan := cmr.FirehoseWithoutReconnect(r.subscriptionID, authToken)

	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Println(err)
			}

			return
		case msg := <-msgChan:
			if msg.GetEventType() == events.Envelope_LogMessage {
				atomic.AddInt64(&r.logCount, 1)
			}

			if msg.GetEventType() == events.Envelope_CounterEvent {
				if msg.GetOrigin() == r.counterOrigin {
					atomic.AddInt64(&r.metricCount, 1)
				}
			}
		}
	}
}
