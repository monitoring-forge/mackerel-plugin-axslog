package axslog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStats(t *testing.T) {
	s := NewStats()
	require.NotNil(t, s, "NewStats() returned nil")
	require.NotNil(t, s.percentiles, "percentiles is nil")
	require.Equal(t, 0, s.percentiles.Count(), "percentiles count = %d; want 0", s.percentiles.Count())
}

func TestStatsAppendAndTotal(t *testing.T) {
	s := NewStats()
	s.Append(0.010, []byte("200"))
	s.Append(0.020, []byte("200"))
	s.Append(0.030, []byte("404"))
	s.Append(0.040, []byte("499"))
	s.Append(0.050, []byte("500"))

	assert.Equal(t, 5.0, s.total, "Total = %f; want 5", s.total)
	assert.Equal(t, 2.0, s.c2xx, "C2xx = %f; want 2", s.c2xx)
	assert.Equal(t, 1.0, s.c4xx, "C4xx = %f; want 1", s.c4xx)
	assert.Equal(t, 1.0, s.c499, "C499 = %f; want 1", s.c499)
	assert.Equal(t, 1.0, s.c5xx, "C5xx = %f; want 1", s.c5xx)
	assert.Equal(t, 5, s.percentiles.Count(), "percentiles count = %d; want 5", s.percentiles.Count())
}

func TestStatsSetDuration(t *testing.T) {
	s := NewStats()
	s.SetDuration(60.0)
	if s.duration != 60.0 {
		t.Errorf("Duration = %f; want 60.0", s.duration)
	}
}

var testPercentileTargets = []PercentileTarget{
	{Name: "50_percentile", Value: 50.0},
	{Name: "90_percentile", Value: 90.0},
	{Name: "99_percentile", Value: 99.0},
	{Name: "99_9_percentile", Value: 99.9},
}

func TestDisplay(t *testing.T) {
	s := NewStats()
	s.Append(0.010, []byte("200"))
	s.Append(0.020, []byte("200"))
	s.Append(0.100, []byte("500"))
	s.SetDuration(60.0)

	require.Equal(t, 3, s.percentiles.Count(), "percentiles count = %d; want 3", s.percentiles.Count())

	output := s.Display(testPercentileTargets, "test")
	assert.Contains(t, output, "axslog.latency_test.average\t0.043333\t")
	assert.Contains(t, output, "axslog.access_num_test.2xx_count\t0.033333\t")
	assert.Contains(t, output, "axslog.access_ratio_test.4xx_percentage\t0.000000\t")
	assert.Contains(t, output, "axslog.latency_test.50_percentile\t0.020000\t")
	assert.Contains(t, output, "axslog.latency_test.90_percentile\t0.084000\t")
	assert.Contains(t, output, "axslog.latency_test.99_percentile\t0.098400\t")
	assert.Contains(t, output, "axslog.latency_test.99_9_percentile\t0.099840\t")
}

func TestDisplayNoData(t *testing.T) {
	s := NewStats()

	output := s.Display(testPercentileTargets, "empty")
	if len(output) != 0 {
		t.Errorf("output should be empty, got: %s", output)
	}
}

func TestDisplayAll(t *testing.T) {
	s1 := NewStats()
	s1.Append(0.010, []byte("200"))
	s1.SetDuration(60.0)

	s2 := NewStats()
	s2.Append(0.020, []byte("404"))
	s2.SetDuration(60.0)

	output := DisplayAll([]*Stats{s1, s2}, testPercentileTargets, "all")
	assert.Contains(t, output, "axslog.latency_all.average\t0.015000\t")
	assert.Contains(t, output, "axslog.access_num_all.2xx_count\t0.016667\t")
	assert.Contains(t, output, "axslog.access_num_all.4xx_count\t0.016667\t")
	assert.Contains(t, output, "axslog.latency_all.50_percentile\t0.015000\t")
	assert.Contains(t, output, "axslog.latency_all.90_percentile\t0.019000\t")
	assert.Contains(t, output, "axslog.latency_all.99_percentile\t0.019900\t")
}

func TestDisplayAllNoDuration(t *testing.T) {
	s := NewStats()
	s.Append(0.010, []byte("200"))

	output := DisplayAll([]*Stats{s}, testPercentileTargets, "noduration")
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
	s.Append(0.001, []byte("100"))
	s.Append(0.002, []byte("200"))
	s.Append(0.003, []byte("301"))
	s.Append(0.004, []byte("404"))
	s.Append(0.005, []byte("499"))
	s.Append(0.006, []byte("503"))

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
	s1.Append(0.010, []byte("200"))
	s1.SetDuration(60.0)

	output := DisplayAll([]*Stats{s1}, testPercentileTargets, "single")
	if !strings.Contains(output, "axslog.access_ratio_single.2xx_percentage\t100.000000\t") {
		t.Errorf("2xx percentage not 100, got: %s", output)
	}
}
