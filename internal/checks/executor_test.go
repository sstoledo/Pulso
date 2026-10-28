package checks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type resolverFunc func(context.Context, string) ([]net.IPAddr, error)

func (f resolverFunc) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return f(ctx, host)
}

func TestExecutorHTTPOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		min        int
		max        int
		wantResult bool
	}{
		{name: "expected status", status: 204, min: 200, max: 399, wantResult: true},
		{name: "unexpected status", status: 500, min: 200, max: 399, wantResult: false},
		{name: "custom expected range", status: 500, min: 500, max: 500, wantResult: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()
			executor := NewExecutor([]string{"127.0.0.1"})

			result := executor.Check(context.Background(), Target{URL: server.URL, TimeoutMS: 1000, ExpectedStatusMin: tt.min, ExpectedStatusMax: tt.max})

			if result.Success != tt.wantResult || result.HTTPStatus == nil || *result.HTTPStatus != tt.status || result.LatencyMS < 0 {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestExecutorTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	executor := NewExecutor([]string{"127.0.0.1"})

	result := executor.Check(context.Background(), Target{URL: server.URL, TimeoutMS: 20, ExpectedStatusMin: 200, ExpectedStatusMax: 399})

	if result.Success || result.ErrorSummary != "request timeout" {
		t.Fatalf("result = %#v", result)
	}
}

func TestExecutorRedirectPolicy(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/allowed":
			w.WriteHeader(http.StatusNoContent)
		case "/redirect-allowed":
			http.Redirect(w, r, "/allowed", http.StatusFound)
		case "/redirect-blocked":
			parsed, _ := url.Parse(serverURL)
			http.Redirect(w, r, "http://localhost:"+parsed.Port()+"/allowed", http.StatusFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	executor := NewExecutor([]string{"127.0.0.1"})
	allowed := executor.Check(context.Background(), Target{URL: server.URL + "/redirect-allowed", TimeoutMS: 1000, ExpectedStatusMin: 200, ExpectedStatusMax: 399})
	if !allowed.Success {
		t.Fatalf("allowed redirect = %#v", allowed)
	}
	blocked := executor.Check(context.Background(), Target{URL: server.URL + "/redirect-blocked", TimeoutMS: 1000, ExpectedStatusMin: 200, ExpectedStatusMax: 399})
	if blocked.Success || blocked.ErrorSummary != "destination blocked" {
		t.Fatalf("blocked redirect = %#v", blocked)
	}
}

func TestExecutorBlocksInternalDestinationsByDefault(t *testing.T) {
	urls := []string{
		"http://127.0.0.1:8080", "http://10.0.0.1:8080", "http://169.254.1.1:8080",
		"http://0.0.0.0:8080", "http://[::1]:8080", "http://[::ffff:127.0.0.1]:8080",
	}
	for _, targetURL := range urls {
		t.Run(targetURL, func(t *testing.T) {
			executor := NewExecutor(nil)
			result := executor.Check(context.Background(), Target{URL: targetURL, TimeoutMS: 100, ExpectedStatusMin: 200, ExpectedStatusMax: 399})
			if result.Success || result.ErrorSummary != "destination blocked" {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestExecutorUsesResolvedIPAndTLSHostname(t *testing.T) {
	server := newExampleTLSServer(t)
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	executor := NewExecutor([]string{"example.test"})
	executor.resolver = resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	})
	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	executor.rootCAs = pool
	target := fmt.Sprintf("https://example.test:%s/health", parsed.Port())

	result := executor.Check(context.Background(), Target{URL: target, TimeoutMS: 1000, ExpectedStatusMin: 200, ExpectedStatusMax: 399})

	if !result.Success {
		t.Fatalf("TLS result = %#v", result)
	}
}

func TestExecutorCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Second)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	executor := NewExecutor([]string{"127.0.0.1"})

	result := executor.Check(ctx, Target{URL: server.URL, TimeoutMS: 5000, ExpectedStatusMin: 200, ExpectedStatusMax: 399})

	if result.Success || result.ErrorSummary != "request canceled" {
		t.Fatalf("result = %#v", result)
	}
}

func newExampleTLSServer(t *testing.T) *httptest.Server {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "example.test"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"example.test"},
	}
	der, err := x509.CreateCertificate(nil, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	server.StartTLS()
	return server
}

func TestBlockedErrorIsInspectable(t *testing.T) {
	executor := NewExecutor(nil)
	_, err := executor.destination(context.Background(), "http://127.0.0.1:8080")
	if !errors.Is(err, ErrDestinationBlocked) {
		t.Fatalf("error = %v", err)
	}
}

func TestErrorSummaryDoesNotIncludeTarget(t *testing.T) {
	executor := NewExecutor(nil)
	result := executor.Check(context.Background(), Target{URL: "http://127.0.0.1:8080/secret", TimeoutMS: 100, ExpectedStatusMin: 200, ExpectedStatusMax: 399})
	if strings.Contains(result.ErrorSummary, "secret") || strings.Contains(result.ErrorSummary, "127.0.0.1") {
		t.Fatalf("unsafe error summary %q", result.ErrorSummary)
	}
}
