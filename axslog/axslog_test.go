package axslog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCode(t *testing.T) {
	tests := []struct {
		status   int
		expected int
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

func TestStatsAppendAndGetTotal(t *testing.T) {
	s := NewStats()
	s.Append(0.010, 200)
	s.Append(0.020, 200)
	s.Append(0.030, 404)
	s.Append(0.040, 499)
	s.Append(0.050, 500)

	total := s.GetTotal()
	assert.Equal(t, 5.0, total, "GetTotal() = %f; want 5", total)
	assert.Equal(t, 2.0, s.c2xx, "c2xx = %f; want 2", s.c2xx)
	assert.Equal(t, 1.0, s.c4xx, "c4xx = %f; want 1", s.c4xx)
	assert.Equal(t, 1.0, s.c499, "c499 = %f; want 1", s.c499)
	assert.Equal(t, 1.0, s.c5xx, "c5xx = %f; want 1", s.c5xx)
	assert.Equal(t, 5, len(s.f64s), "len(f64s) = %d; want 5", len(s.f64s))
}

func TestStatsSetDuration(t *testing.T) {
	s := NewStats()
	s.SetDuration(60.0)
	if s.duration != 60.0 {
		t.Errorf("duration = %f; want 60.0", s.duration)
	}
}

func TestBFloat64(t *testing.T) {
	v, err := BFloat64([]byte("3.14"))
	if err != nil {
		t.Fatalf("BFloat64 error: %v", err)
	}
	if v != 3.14 {
		t.Errorf("BFloat64 = %f; want 3.14", v)
	}
}

func TestBFloat64Invalid(t *testing.T) {
	_, err := BFloat64([]byte("not-a-number"))
	if err == nil {
		t.Error("BFloat64 should return error for invalid input")
	}
}

func TestBInt(t *testing.T) {
	v, err := BInt([]byte("200"))
	if err != nil {
		t.Fatalf("BInt error: %v", err)
	}
	if v != 200 {
		t.Errorf("BInt = %d; want 200", v)
	}
}

func TestBIntInvalid(t *testing.T) {
	_, err := BInt([]byte("abc"))
	if err == nil {
		t.Error("BInt should return error for invalid input")
	}
}

func TestHumanBytesUnmarshalFlag(t *testing.T) {
	var hb HumanBytes
	if err := hb.UnmarshalFlag("10MB"); err != nil {
		t.Fatalf("UnmarshalFlag error: %v", err)
	}
	if hb != 10*1000*1000 {
		t.Errorf("HumanBytes = %d; want 10000000", hb)
	}
}

func TestHumanBytesUnmarshalFlagInvalid(t *testing.T) {
	var hb HumanBytes
	if err := hb.UnmarshalFlag("invalid"); err == nil {
		t.Error("UnmarshalFlag should return error for invalid input")
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

	assert.Equal(t, 6.0, s.total, "total = %f; want 6", s.total)
	assert.Equal(t, 1.0, s.c1xx, "c1xx = %f; want 1", s.c1xx)
	assert.Equal(t, 1.0, s.c2xx, "c2xx = %f; want 1", s.c2xx)
	assert.Equal(t, 1.0, s.c3xx, "c3xx = %f; want 1", s.c3xx)
	assert.Equal(t, 1.0, s.c4xx, "c4xx = %f; want 1", s.c4xx)
	assert.Equal(t, 1.0, s.c499, "c499 = %f; want 1", s.c499)
	assert.Equal(t, 1.0, s.c5xx, "c5xx = %f; want 1", s.c5xx)
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
