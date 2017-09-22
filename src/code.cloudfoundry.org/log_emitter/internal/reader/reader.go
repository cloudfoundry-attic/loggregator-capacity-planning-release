package reader

import (
	"bytes"
	"crypto/tls"
	"log"
	"sync/atomic"
	"time"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"

	"code.cloudfoundry.org/authenticator"
)

type Reader struct {
	dopplerAddr  string
	appID        string
	logMsg       string
	auth         *authenticator.Authenticator
	tlsConfig    *tls.Config
	receivedMsgs int64
}

func New(
	dopplerAddr string,
	appID string,
	logMsg string,
	a *authenticator.Authenticator,
) *Reader {
	return &Reader{
		dopplerAddr: dopplerAddr,
		appID:       appID,
		auth:        a,
		logMsg:      logMsg,
		tlsConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}

func (r *Reader) Count() int64 {
	return atomic.SwapInt64(&r.receivedMsgs, 0)
}

func (r *Reader) Run() {
	for {
		token, err := r.auth.Token()
		if err != nil {
			log.Printf("failed to authenticate with UAA: %s", err)
			time.Sleep(time.Second)
			continue
		}

		r.readLogs(token)
	}
}

func (r *Reader) readLogs(authToken string) {
	cmr := consumer.New(r.dopplerAddr, r.tlsConfig, nil)

	msgChan, errChan := cmr.Stream(r.appID, authToken)

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
			log := msg.GetLogMessage()
			if bytes.Contains(log.GetMessage(), []byte(r.logMsg)) {
				atomic.AddInt64(&r.receivedMsgs, 1)
			}
		}
	}
}
