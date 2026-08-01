package main

import (
	"testing"

	"github.com/monitoring-forge/mackerel-plugin-axslog/axslog"
)

func TestFiltered(t *testing.T) {
	opt := &axslog.Opt{
		Filter:       "test",
		InvertFilter: false,
	}
	stats := axslog.NewStats()
	p := NewParser(opt, stats)

	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"Contains filter", []byte("This is a test log line."), true},
		{"Does not contain filter", []byte("This is a log line."), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.filtered(tt.input)
			if result != tt.expected {
				t.Errorf("filtered() = %v, want %v", result, tt.expected)
			}
		})
	}
}
