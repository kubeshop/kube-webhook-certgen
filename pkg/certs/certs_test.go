package certs

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func handler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "Hello World")
}

func TestCertificateCreation(t *testing.T) {
	t.Parallel()

	ca, cert, key, err := GenerateCerts("localhost")
	assert.NoError(t, err)

	c, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Fatal(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    caCertPool,
			ServerName: "localhost",
			MinVersion: tls.VersionTLS12,
		},
	}

	ts := httptest.NewUnstartedServer(http.HandlerFunc(handler))
	ts.TLS = &tls.Config{Certificates: []tls.Certificate{c}, MinVersion: tls.VersionTLS12}
	ts.StartTLS()
	defer ts.Close()

	ctx := context.Background()

	client := &http.Client{Transport: tr}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("response code was %v; want 200", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte("Hello World")

	if !bytes.Equal(expected, body) {
		t.Errorf("response body was '%v'; want '%v'", expected, body)
	}
}
