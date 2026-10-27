package monitors

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

type CreateInput struct {
	Name              string `json:"name"`
	URL               string `json:"url"`
	IntervalSeconds   int    `json:"interval_seconds"`
	TimeoutMS         int    `json:"timeout_ms"`
	ExpectedStatusMin int    `json:"expected_status_min"`
	ExpectedStatusMax int    `json:"expected_status_max"`
	ShowPublic        bool   `json:"show_public"`
}

func NormalizeCreate(input CreateInput) (CreateInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	if input.IntervalSeconds == 0 {
		input.IntervalSeconds = 60
	}
	if input.TimeoutMS == 0 {
		input.TimeoutMS = 5000
	}
	if input.ExpectedStatusMin == 0 {
		input.ExpectedStatusMin = 200
	}
	if input.ExpectedStatusMax == 0 {
		input.ExpectedStatusMax = 399
	}
	if err := validateConfiguration(input.Name, input.URL, input.IntervalSeconds, input.TimeoutMS, input.ExpectedStatusMin, input.ExpectedStatusMax); err != nil {
		return CreateInput{}, err
	}
	return input, nil
}

func validateConfiguration(name, targetURL string, intervalSeconds, timeoutMS, statusMin, statusMax int) error {
	fields := make(map[string]string)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		fields["name"] = "Enter between 1 and 100 characters."
	}
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.Fragment != "" {
		fields["url"] = "Enter an HTTP(S) URL without credentials or fragments."
	}
	if intervalSeconds != 30 && intervalSeconds != 60 && intervalSeconds != 300 {
		fields["interval_seconds"] = "Choose 30, 60, or 300 seconds."
	}
	if timeoutMS < 100 || timeoutMS > 10000 {
		fields["timeout_ms"] = "Enter between 100 and 10,000 milliseconds."
	}
	if statusMin < 100 || statusMin > 599 {
		fields["expected_status_min"] = "Enter a status between 100 and 599."
	}
	if statusMax < 100 || statusMax > 599 {
		fields["expected_status_max"] = "Enter a status between 100 and 599."
	} else if statusMin >= 100 && statusMin <= 599 && statusMin > statusMax {
		fields["expected_status_max"] = "Maximum status must be greater than or equal to minimum status."
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}
