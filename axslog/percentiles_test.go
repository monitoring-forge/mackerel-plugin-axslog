package axslog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPercentileTargetFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected PercentileTarget
		hasError bool
	}{
		{"50", PercentileTarget{Name: "50_percentile", Value: 50.0}, false},
		{"90", PercentileTarget{Name: "90_percentile", Value: 90.0}, false},
		{"99", PercentileTarget{Name: "99_percentile", Value: 99.0}, false},
		{"99.9", PercentileTarget{Name: "99_9_percentile", Value: 99.9}, false},
		{"invalid", PercentileTarget{}, true},
		{"", PercentileTarget{}, true},
		{"100", PercentileTarget{Name: "100_percentile", Value: 100.0}, false},
		{"100.0", PercentileTarget{Name: "100_percentile", Value: 100.0}, false},
		{"100.1", PercentileTarget{}, true},
		{"1000", PercentileTarget{}, true},
		{"-1", PercentileTarget{}, true},
		{"0", PercentileTarget{Name: "0_percentile", Value: 0.0}, false},
	}

	for _, tt := range tests {
		got, err := PercentileTargetFromString(tt.input)
		if tt.hasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, tt.expected, got)
	}

}

func TestPercentileTargetsFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected []PercentileTarget
		hasError bool
	}{
		{"50,90,99", []PercentileTarget{
			{Name: "50_percentile", Value: 50.0},
			{Name: "90_percentile", Value: 90.0},
			{Name: "99_percentile", Value: 99.0},
		}, false},
		{"99.9,100", []PercentileTarget{
			{Name: "99_9_percentile", Value: 99.9},
			{Name: "100_percentile", Value: 100.0},
		}, false},
		{"invalid,50", nil, true},
		{"", nil, true},
	}

	for _, tt := range tests {
		got, err := PercentileTargetsFromString(tt.input)
		if tt.hasError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, tt.expected, got)
	}
}
