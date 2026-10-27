package monitors_test

import (
	"errors"
	"strings"
	"testing"

	"pulso/internal/monitors"
)

func TestNormalizeCreateAppliesDefaults(t *testing.T) {
	input, err := monitors.NormalizeCreate(monitors.CreateInput{
		Name: "  Public API  ", URL: " https://example.com/health ",
	})
	if err != nil {
		t.Fatalf("NormalizeCreate() error = %v", err)
	}
	if input.Name != "Public API" || input.URL != "https://example.com/health" {
		t.Fatalf("normalized input = %#v", input)
	}
	if input.IntervalSeconds != 60 || input.TimeoutMS != 5000 || input.ExpectedStatusMin != 200 || input.ExpectedStatusMax != 399 {
		t.Fatalf("defaults = %#v", input)
	}
}

func TestNormalizeCreateRejectsInvalidConfiguration(t *testing.T) {
	valid := func() monitors.CreateInput {
		return monitors.CreateInput{Name: "API", URL: "https://example.com", IntervalSeconds: 60, TimeoutMS: 5000, ExpectedStatusMin: 200, ExpectedStatusMax: 399}
	}
	tests := []struct {
		name   string
		mutate func(*monitors.CreateInput)
		field  string
	}{
		{name: "empty name", mutate: func(i *monitors.CreateInput) { i.Name = " " }, field: "name"},
		{name: "long name", mutate: func(i *monitors.CreateInput) { i.Name = strings.Repeat("a", 101) }, field: "name"},
		{name: "unsupported scheme", mutate: func(i *monitors.CreateInput) { i.URL = "ftp://example.com" }, field: "url"},
		{name: "missing host", mutate: func(i *monitors.CreateInput) { i.URL = "https://" }, field: "url"},
		{name: "credentials", mutate: func(i *monitors.CreateInput) { i.URL = "https://user:secret@example.com" }, field: "url"},
		{name: "fragment", mutate: func(i *monitors.CreateInput) { i.URL = "https://example.com/#secret" }, field: "url"},
		{name: "interval", mutate: func(i *monitors.CreateInput) { i.IntervalSeconds = 29 }, field: "interval_seconds"},
		{name: "timeout below minimum", mutate: func(i *monitors.CreateInput) { i.TimeoutMS = 99 }, field: "timeout_ms"},
		{name: "timeout above maximum", mutate: func(i *monitors.CreateInput) { i.TimeoutMS = 10001 }, field: "timeout_ms"},
		{name: "status minimum", mutate: func(i *monitors.CreateInput) { i.ExpectedStatusMin = 99 }, field: "expected_status_min"},
		{name: "status maximum", mutate: func(i *monitors.CreateInput) { i.ExpectedStatusMax = 600 }, field: "expected_status_max"},
		{name: "reversed status range", mutate: func(i *monitors.CreateInput) { i.ExpectedStatusMin = 400; i.ExpectedStatusMax = 399 }, field: "expected_status_max"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := valid()
			tt.mutate(&input)
			_, err := monitors.NormalizeCreate(input)
			var validation *monitors.ValidationError
			if !errors.As(err, &validation) || validation.Fields[tt.field] == "" {
				t.Fatalf("error = %#v, missing field %q", err, tt.field)
			}
		})
	}
}
