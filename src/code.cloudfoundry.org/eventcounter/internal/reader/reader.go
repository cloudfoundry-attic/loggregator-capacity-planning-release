package reader

import (
	"crypto/tls"
	"log"
	"sync/atomic"
	"time"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"

	"code.cloudfoundry.org/eventcounter/internal/authenticator"
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

func (r *Reader) SwapCounts() (logCount int64, metricCount int64) {
	logCount = atomic.SwapInt64(&r.logCount, 0)
	metricCount = atomic.SwapInt64(&r.metricCount, 0)

	return logCount, metricCount
}

func (r *Reader) read(authToken string) {
	cmr := consumer.New(r.egressAddr, r.tlsConfig, nil)

	msgChan, errChan := cmr.Firehose(r.subscriptionID, authToken)

	go func() {
		for err := range errChan {
			if err == nil {
				return
			}

			log.Println(err)
		}
	}()

	for msg := range msgChan {
		if msg == nil {
			return
		}

		if msg.GetEventType() == events.Envelope_LogMessage {
			atomic.AddInt64(&r.logCount, 1)
			return
		}

		if msg.GetEventType() == events.Envelope_CounterEvent {
			if msg.GetOrigin() == r.counterOrigin {
				atomic.AddInt64(&r.metricCount, 1)
			}
			return
		}
	}
}
