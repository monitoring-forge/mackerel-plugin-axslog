package axslog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCode(t *testing.T) {
	tests := []struct {
		status   int64
		expected int64
	}{
		{200, 2},
		{301, 3},
		{404, 4},
		{499, 499},
		{500, 5},
		{100, 1},
	}
	for _, tt := range tests {
		if got := statusCode(tt.status); got != tt.expected {
			t.Errorf("statusCode(%d) = %d; want %d", tt.status, got, tt.expected)
		}
	}
}

func TestNewStats(t *testing.T) {
	s := NewStats()
	require.NotNil(t, s, "NewStats() returned nil")
	require.NotNil(t, s.f64s, "f64s slice is nil")
	require.Equal(t, 0, len(s.f64s), "f64s length = %d; want 0", len(s.f64s))
}

func TestStatsAppendAndTotal(t *testing.T) {
	s := NewStats()
	s.Append(0.010, 200)
	s.Append(0.020, 200)
	s.Append(0.030, 404)
	s.Append(0.040, 499)
	s.Append(0.050, 500)

	assert.Equal(t, 5.0, s.total, "Total = %f; want 5", s.total)
	assert.Equal(t, 2.0, s.c2xx, "C2xx = %f; want 2", s.c2xx)
	assert.Equal(t, 1.0, s.c4xx, "C4xx = %f; want 1", s.c4xx)
	assert.Equal(t, 1.0, s.c499, "C499 = %f; want 1", s.c499)
	assert.Equal(t, 1.0, s.c5xx, "C5xx = %f; want 1", s.c5xx)
	assert.Equal(t, 5, len(s.f64s), "len(f64s) = %d; want 5", len(s.f64s))
}

func TestStatsSetDuration(t *testing.T) {
	s := NewStats()
	s.SetDuration(60.0)
	if s.duration != 60.0 {
		t.Errorf("Duration = %f; want 60.0", s.duration)
	}
}

func TestDisplay(t *testing.T) {
	s := NewStats()
	s.Append(0.010, 200)
	s.Append(0.020, 200)
	s.Append(0.100, 500)
	s.SetDuration(60.0)

	output := s.Display("test")
	assert.Contains(t, output, "axslog.latency_test.average")
	assert.Contains(t, output, "axslog.access_num_test.2xx_count")
	assert.Contains(t, output, "axslog.access_ratio_test.5xx_percentage")
}

func TestDisplayNoData(t *testing.T) {
	s := NewStats()

	output := s.Display("empty")
	if len(output) != 0 {
		t.Errorf("output should be empty, got: %s", output)
	}
}

func TestDisplayAll(t *testing.T) {
	s1 := NewStats()
	s1.Append(0.010, 200)
	s1.SetDuration(60.0)

	s2 := NewStats()
	s2.Append(0.020, 404)
	s2.SetDuration(60.0)

	output := DisplayAll([]*Stats{s1, s2}, "all")
	assert.Contains(t, output, "axslog.latency_all.average")
	assert.Contains(t, output, "axslog.access_num_all.2xx_count")
	assert.Contains(t, output, "axslog.access_num_all.4xx_count")
}

func TestDisplayAllNoDuration(t *testing.T) {
	s := NewStats()
	s.Append(0.010, 200)

	output := DisplayAll([]*Stats{s}, "noduration")
	if strings.Contains(output, "axslog.access_num_") {
		t.Error("output should not contain access_num when duration is zero")
	}
}

func TestFlags(t *testing.T) {
	if PtimeFlag != 1 {
		t.Errorf("PtimeFlag = %d; want 1", PtimeFlag)
	}
	if StatusFlag != 2 {
		t.Errorf("StatusFlag = %d; want 2", StatusFlag)
	}
	if AllFlagOK != 3 {
		t.Errorf("AllFlagOK = %d; want 3", AllFlagOK)
	}
}

func TestStatsAppendAllStatusClasses(t *testing.T) {
	s := NewStats()
	s.Append(0.001, 100)
	s.Append(0.002, 200)
	s.Append(0.003, 301)
	s.Append(0.004, 404)
	s.Append(0.005, 499)
	s.Append(0.006, 503)

	assert.Equal(t, 6.0, s.total, "Total = %f; want 6", s.total)
	assert.Equal(t, 1.0, s.c1xx, "C1xx = %f; want 1", s.c1xx)
	assert.Equal(t, 1.0, s.c2xx, "C2xx = %f; want 1", s.c2xx)
	assert.Equal(t, 1.0, s.c3xx, "C3xx = %f; want 1", s.c3xx)
	assert.Equal(t, 1.0, s.c4xx, "C4xx = %f; want 1", s.c4xx)
	assert.Equal(t, 1.0, s.c499, "C499 = %f; want 1", s.c499)
	assert.Equal(t, 1.0, s.c5xx, "C5xx = %f; want 1", s.c5xx)
}

func TestDisplayAllAggregatedPercentages(t *testing.T) {
	s1 := NewStats()
	s1.Append(0.010, 200)
	s1.SetDuration(60.0)

	output := DisplayAll([]*Stats{s1}, "single")
	if !strings.Contains(output, "axslog.access_ratio_single.2xx_percentage\t100.000000\t") {
		t.Errorf("2xx percentage not 100, got: %s", output)
	}
}
