package checks

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

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

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Check(ctx context.Context, target Target) Result {
	started := time.Now()
	if target.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(target.TimeoutMS)*time.Millisecond)
		defer cancel()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		return Result{LatencyMS: elapsedMS(started), ErrorSummary: summaryForError(ctx, err)}
	}
	request.Header.Set("User-Agent", "Pulso/1.0")
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := client.Do(request)
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
