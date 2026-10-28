package checks

import "testing"

func TestApplyResult(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		failures   int
		success    bool
		wantStatus string
		wantFails  int
	}{
		{name: "first success", status: "UNKNOWN", success: true, wantStatus: "UP"},
		{name: "first failure remains unknown", status: "UNKNOWN", wantStatus: "UNKNOWN", wantFails: 1},
		{name: "first failure preserves up", status: "UP", wantStatus: "UP", wantFails: 1},
		{name: "second failure is down", status: "UP", failures: 1, wantStatus: "DOWN", wantFails: 2},
		{name: "failure while down", status: "DOWN", failures: 2, wantStatus: "DOWN", wantFails: 3},
		{name: "recovery", status: "DOWN", failures: 3, success: true, wantStatus: "UP", wantFails: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, failures := ApplyResult(tt.status, tt.failures, tt.success)
			if status != tt.wantStatus || failures != tt.wantFails {
				t.Fatalf("got %s/%d, want %s/%d", status, failures, tt.wantStatus, tt.wantFails)
			}
		})
	}
}
