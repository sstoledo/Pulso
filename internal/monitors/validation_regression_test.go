package monitors_test

import (
	"errors"
	"testing"

	"pulso/internal/monitors"
)

func TestNormalizeCreateRejectsEmptyURLFragment(t *testing.T) {
	_, err := monitors.NormalizeCreate(monitors.CreateInput{
		Name: "API", URL: "https://example.com/#", IntervalSeconds: 60, TimeoutMS: 5000,
		ExpectedStatusMin: 200, ExpectedStatusMax: 399,
	})
	var validation *monitors.ValidationError
	if !errors.As(err, &validation) || validation.Fields["url"] == "" {
		t.Fatalf("error = %#v, missing URL validation", err)
	}
}
