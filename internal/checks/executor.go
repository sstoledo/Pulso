package checks

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrDestinationBlocked = errors.New("destination blocked")

var blockedPrefixes = []net.IPNet{
	mustCIDR("0.0.0.0/8"),
	mustCIDR("100.64.0.0/10"),
	mustCIDR("192.0.0.0/24"),
	mustCIDR("192.0.2.0/24"),
	mustCIDR("198.18.0.0/15"),
	mustCIDR("198.51.100.0/24"),
	mustCIDR("203.0.113.0/24"),
	mustCIDR("240.0.0.0/4"),
}

type Target struct {
	URL               string
	TimeoutMS         int
	ExpectedStatusMin int
	ExpectedStatusMax int
}

type Result struct {
	Success      bool
	HTTPStatus   *int
	LatencyMS    int
	ErrorSummary string
}

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type Executor struct {
	resolver     Resolver
	allowPrivate map[string]bool
	rootCAs      *x509.CertPool
}

type destination struct {
	ip       net.IP
	port     string
	hostname string
}

func NewExecutor(allowedPrivateHosts []string) *Executor {
	allowed := make(map[string]bool, len(allowedPrivateHosts))
	for _, host := range allowedPrivateHosts {
		allowed[normalizeHost(host)] = true
	}
	return &Executor{resolver: net.DefaultResolver, allowPrivate: allowed}
}

func (e *Executor) Check(ctx context.Context, target Target) Result {
	started := time.Now()
	if target.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(target.TimeoutMS)*time.Millisecond)
		defer cancel()
	}
	response, err := e.executeOnce(ctx, target.URL)
	if err != nil {
		return Result{LatencyMS: elapsedMS(started), ErrorSummary: summaryForError(ctx, err)}
	}
	status := response.StatusCode
	if err := drainAndClose(response); err != nil {
		return Result{HTTPStatus: &status, LatencyMS: elapsedMS(started), ErrorSummary: summaryForError(ctx, err)}
	}
	success := status >= target.ExpectedStatusMin && status <= target.ExpectedStatusMax
	result := Result{Success: success, HTTPStatus: &status, LatencyMS: elapsedMS(started)}
	if !success {
		result.ErrorSummary = "unexpected HTTP status"
	}
	return result
}

func (e *Executor) executeOnce(ctx context.Context, targetURL string) (*http.Response, error) {
	destination, err := e.destination(ctx, targetURL)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Pulso/1.0")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(destination.ip.String(), destination.port))
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.ServerName = destination.hostname
	if e.rootCAs != nil {
		transport.TLSClientConfig.RootCAs = e.rootCAs
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client.Do(request)
}

func (e *Executor) destination(ctx context.Context, rawURL string) (destination, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || strings.Contains(rawURL, "#") {
		return destination{}, ErrDestinationBlocked
	}
	hostname := normalizeHost(parsed.Hostname())
	if hostname == "" {
		return destination{}, ErrDestinationBlocked
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	addresses := []net.IPAddr{}
	if ip := net.ParseIP(hostname); ip != nil {
		addresses = append(addresses, net.IPAddr{IP: ip})
	} else {
		addresses, err = e.resolver.LookupIPAddr(ctx, hostname)
		if err != nil {
			return destination{}, err
		}
	}
	if len(addresses) == 0 {
		return destination{}, ErrDestinationBlocked
	}
	if !e.allowPrivate[hostname] {
		for _, address := range addresses {
			if blockedIP(address.IP) {
				return destination{}, ErrDestinationBlocked
			}
		}
	}
	return destination{ip: addresses[0].IP, port: port, hostname: hostname}, nil
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func mustCIDR(value string) net.IPNet {
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		panic(err)
	}
	return *network
}

func normalizeHost(host string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
}

func drainAndClose(response *http.Response) error {
	_, err := io.Copy(io.Discard, io.LimitReader(response.Body, 32*1024))
	closeErr := response.Body.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func elapsedMS(started time.Time) int {
	elapsed := time.Since(started).Milliseconds()
	if elapsed < 0 {
		return 0
	}
	return int(elapsed)
}

func summaryForError(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, ErrDestinationBlocked):
		return "destination blocked"
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "request timeout"
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		return "request canceled"
	case errors.Is(err, &net.DNSError{}):
		return "name resolution failed"
	default:
		return "request failed"
	}
}
