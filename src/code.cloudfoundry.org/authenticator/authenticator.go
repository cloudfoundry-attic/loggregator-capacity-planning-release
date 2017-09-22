package authenticator

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type Authenticator struct {
	clientID     string
	clientSecret string
	uaaAddr      string
	httpClient   httpClient
}

func New(id, secret, uaaAddr string, opts ...authenticatorOpt) *Authenticator {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{},
			DisableKeepAlives: true,
		},
	}

	a := &Authenticator{
		clientID:     id,
		clientSecret: secret,
		uaaAddr:      uaaAddr,
		httpClient:   httpClient,
	}

	for _, o := range opts {
		o(a)
	}

	return a
}

func (a *Authenticator) Token() (string, error) {
	response, err := a.httpClient.PostForm(a.uaaAddr+"/oauth/token", url.Values{
		"response_type": {"token"},
		"grant_type":    {"client_credentials"},
		"client_id":     {a.clientID},
		"client_secret": {a.clientSecret},
	})
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Expected 200 status code from /oauth/token, got %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}

	oauthResponse := make(map[string]interface{})
	err = json.Unmarshal(body, &oauthResponse)
	if err != nil {
		return "", err
	}

	accessTokenInterface, ok := oauthResponse["access_token"]
	if !ok {
		return "", errors.New("No access_token on UAA oauth response")
	}

	accessToken, ok := accessTokenInterface.(string)
	if !ok {
		return "", errors.New("access_token on UAA oauth response not a string")
	}

	return "bearer " + accessToken, nil
}

type httpClient interface {
	PostForm(string, url.Values) (*http.Response, error)
}

type authenticatorOpt func(*Authenticator)

func WithHTTPClient(c httpClient) authenticatorOpt {
	return func(a *Authenticator) {
		a.httpClient = c
	}
}
