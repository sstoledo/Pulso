package incidents_test

import (
	"testing"

	"pulso/internal/incidents"
)

func TestTransitionActions(t *testing.T) {
	tests := []struct {
		name      string
		failures  int
		hasOpen   bool
		success   bool
		wantOpen  bool
		wantClose bool
	}{
		{name: "first failure", failures: 1, wantOpen: false, wantClose: false},
		{name: "second failure opens", failures: 2, wantOpen: true, wantClose: false},
		{name: "existing open incident", failures: 3, hasOpen: true, wantOpen: false, wantClose: false},
		{name: "recovery closes", failures: 0, hasOpen: true, success: true, wantOpen: false, wantClose: true},
		{name: "success without incident", failures: 0, hasOpen: false, success: true, wantOpen: false, wantClose: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := incidents.Action(tt.failures, tt.hasOpen, tt.success)
			if action.Open != tt.wantOpen || action.Close != tt.wantClose {
				t.Fatalf("action = %#v, want open=%v close=%v", action, tt.wantOpen, tt.wantClose)
			}
		})
	}
}
