package model

import (
	"testing"
)

func TestRiskLevelString(t *testing.T) {
	tests := []struct {
		level RiskLevel
		want  string
	}{
		{RiskInfo, "info"},
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskCritical, "critical"},
		{RiskLevel(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}
