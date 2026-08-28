package axslog

import (
	"bytes"
	"fmt"
	"time"

	"github.com/montanaflynn/stats"
)

var (
	// PtimeFlag : ptime is exists
	PtimeFlag = 1
	// StatusFlag : stattus is exists
	StatusFlag = 2
	// AllFlagOK : all OK
	AllFlagOK = 3
)

// Reader :
type Reader interface {
	Parse([]byte) (int, []byte, []byte)
}

// Stats :
type Stats struct {
	f64s     []float64
	c1xx     float64
	c2xx     float64
	c3xx     float64
	c4xx     float64
	c499     float64
	c5xx     float64
	total    float64
	duration float64
}

// StatsCh :
type StatsCh struct {
	Stats   *Stats
	Logfile string
	Err     error
}

func statusCode(status int64) int64 {
	switch status {
	case 499:
		return 499
	default:
		return status / 100
	}
}

// NewStats :
func NewStats() *Stats {
	f64s := make([]float64, 0)
	return &Stats{
		f64s: f64s,
	}
}

// Dump status for debug/test
func (s *Stats) Dump() map[string]float64 {
	return map[string]float64{
		"c1xx":  s.c1xx,
		"c2xx":  s.c2xx,
		"c3xx":  s.c3xx,
		"c4xx":  s.c4xx,
		"c499":  s.c499,
		"c5xx":  s.c5xx,
		"total": s.total,
	}
}

// Append :
func (s *Stats) Append(ptime float64, status int64) {
	switch statusCode(status) {
	case 2:
		s.c2xx++
	case 3:
		s.c3xx++
	case 4:
		s.c4xx++
	case 5:
		s.c5xx++
	case 499:
		s.c499++
	case 1:
		s.c1xx++
	}
	s.total++

	s.f64s = append(s.f64s, ptime)
}

// SetDuration :
func (s *Stats) SetDuration(d float64) {
	s.duration = d
}

// Display :
func (s *Stats) Display(keyPrefix string) string {
	var buf bytes.Buffer
	now := uint64(time.Now().Unix())
	// fmt.Printf("count: %d\n", len(f64s))
	if len(s.f64s) > 0 {
		mean, _ := stats.Mean(s.f64s)
		fmt.Fprintf(&buf, "axslog.latency_%s.average\t%f\t%d\n", keyPrefix, mean, now)
		p99, _ := stats.Percentile(s.f64s, 99)
		fmt.Fprintf(&buf, "axslog.latency_%s.99_percentile\t%f\t%d\n", keyPrefix, p99, now)
		p95, _ := stats.Percentile(s.f64s, 95)
		fmt.Fprintf(&buf, "axslog.latency_%s.95_percentile\t%f\t%d\n", keyPrefix, p95, now)
		p90, _ := stats.Percentile(s.f64s, 90)
		fmt.Fprintf(&buf, "axslog.latency_%s.90_percentile\t%f\t%d\n", keyPrefix, p90, now)
	}

	if s.duration > 0 {
		fmt.Fprintf(&buf, "axslog.access_num_%s.1xx_count\t%f\t%d\n", keyPrefix, s.c1xx/s.duration, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.2xx_count\t%f\t%d\n", keyPrefix, s.c2xx/s.duration, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.3xx_count\t%f\t%d\n", keyPrefix, s.c3xx/s.duration, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.4xx_count\t%f\t%d\n", keyPrefix, s.c4xx/s.duration, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.499_count\t%f\t%d\n", keyPrefix, s.c499/s.duration, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.5xx_count\t%f\t%d\n", keyPrefix, s.c5xx/s.duration, now)
		fmt.Fprintf(&buf, "axslog.access_total_%s.count\t%f\t%d\n", keyPrefix, s.total/s.duration, now)
	}
	if s.total > 0 {
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.1xx_percentage\t%f\t%d\n", keyPrefix, s.c1xx*100/s.total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.2xx_percentage\t%f\t%d\n", keyPrefix, s.c2xx*100/s.total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.3xx_percentage\t%f\t%d\n", keyPrefix, s.c3xx*100/s.total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.4xx_percentage\t%f\t%d\n", keyPrefix, s.c4xx*100/s.total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.499_percentage\t%f\t%d\n", keyPrefix, s.c499*100/s.total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.5xx_percentage\t%f\t%d\n", keyPrefix, s.c5xx*100/s.total, now)
	}
	return buf.String()
}

// DisplayAll :
func DisplayAll(statsAll []*Stats, keyPrefix string) string {
	var buf bytes.Buffer
	now := uint64(time.Now().Unix())

	f64s := make([]float64, 0)
	c1xx := float64(0)
	c2xx := float64(0)
	c3xx := float64(0)
	c4xx := float64(0)
	c499 := float64(0)
	c5xx := float64(0)
	total := float64(0)
	allDurationNG := true
	for _, s := range statsAll {
		f64s = append(f64s, s.f64s...)
		if s.duration > 0 {
			allDurationNG = false
			c1xx += s.c1xx / s.duration
			c2xx += s.c2xx / s.duration
			c3xx += s.c3xx / s.duration
			c4xx += s.c4xx / s.duration
			c499 += s.c499 / s.duration
			c5xx += s.c5xx / s.duration
			total += s.total / s.duration
		}
	}
	// fmt.Printf("count: %d\n", len(f64s))
	if len(f64s) > 0 {
		mean, _ := stats.Mean(f64s)
		fmt.Fprintf(&buf, "axslog.latency_%s.average\t%f\t%d\n", keyPrefix, mean, now)
		p99, _ := stats.Percentile(f64s, 99)
		fmt.Fprintf(&buf, "axslog.latency_%s.99_percentile\t%f\t%d\n", keyPrefix, p99, now)
		p95, _ := stats.Percentile(f64s, 95)
		fmt.Fprintf(&buf, "axslog.latency_%s.95_percentile\t%f\t%d\n", keyPrefix, p95, now)
		p90, _ := stats.Percentile(f64s, 90)
		fmt.Fprintf(&buf, "axslog.latency_%s.90_percentile\t%f\t%d\n", keyPrefix, p90, now)
	}

	if !allDurationNG {
		fmt.Fprintf(&buf, "axslog.access_num_%s.1xx_count\t%f\t%d\n", keyPrefix, c1xx, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.2xx_count\t%f\t%d\n", keyPrefix, c2xx, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.3xx_count\t%f\t%d\n", keyPrefix, c3xx, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.4xx_count\t%f\t%d\n", keyPrefix, c4xx, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.499_count\t%f\t%d\n", keyPrefix, c499, now)
		fmt.Fprintf(&buf, "axslog.access_num_%s.5xx_count\t%f\t%d\n", keyPrefix, c5xx, now)
		fmt.Fprintf(&buf, "axslog.access_total_%s.count\t%f\t%d\n", keyPrefix, total, now)
	}

	if total > 0 {
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.1xx_percentage\t%f\t%d\n", keyPrefix, c1xx*100/total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.2xx_percentage\t%f\t%d\n", keyPrefix, c2xx*100/total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.3xx_percentage\t%f\t%d\n", keyPrefix, c3xx*100/total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.4xx_percentage\t%f\t%d\n", keyPrefix, c4xx*100/total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.499_percentage\t%f\t%d\n", keyPrefix, c499*100/total, now)
		fmt.Fprintf(&buf, "axslog.access_ratio_%s.5xx_percentage\t%f\t%d\n", keyPrefix, c5xx*100/total, now)
	}

	return buf.String()
}
