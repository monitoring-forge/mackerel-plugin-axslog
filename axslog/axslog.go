package axslog

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/monitoring-forge/sampdo"
)

// Reader defines the interface for parsing log entries to extract processing time and status code.
type Reader interface {
	Parse([]byte) (ptime []byte, status []byte)
}

// Stats holds the aggregated statistics for HTTP status codes and request processing times.
type Stats struct {
	percentiles *sampdo.Sampdo
	c1xx        float64
	c2xx        float64
	c3xx        float64
	c4xx        float64
	c499        float64
	c5xx        float64
	total       float64
	duration    float64
}

// StatsCh holds a single statistics snapshot along with the associated log file and any error encountered.
type StatsCh struct {
	Stats   *Stats
	Logfile string
	Err     error
}

// NewStats creates and returns a new Stats instance with initialized percentiles.
func NewStats() *Stats {
	sampdo := sampdo.New(sampdo.WithInitialCapacity(1024))
	return &Stats{
		percentiles: sampdo,
	}
}

// Dump returns a map of the current status counts for debugging or testing purposes.
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

// Append updates the statistics with a new request's processing time and status code.
func (s *Stats) Append(ptime float64, status []byte) {
	if bytes.Equal(status, []byte("499")) {
		s.c499++
	} else if len(status) > 0 {
		switch status[0] {
		case '1':
			s.c1xx++
		case '2':
			s.c2xx++
		case '3':
			s.c3xx++
		case '4':
			s.c4xx++
		case '5':
			s.c5xx++
		}
	}

	s.total++

	if s.percentiles == nil {
		s.percentiles = sampdo.New(sampdo.WithInitialCapacity(1024))
	}
	err := s.percentiles.Append(ptime)
	if err != nil {
		log.Printf("error appending percentile: %v\n", err)
	}
}

// SetDuration sets the duration over which the statistics were collected.
func (s *Stats) SetDuration(d float64) {
	s.duration = d
}

func displayPercentiles(w io.Writer, percentile *sampdo.Sampdo, percentileTargets []PercentileTarget, keyPrefix string, now uint64) error {
	if percentile == nil || percentile.Count() == 0 {
		return nil
	}
	sorted, err := percentile.Sorted()
	if err != nil {
		return fmt.Errorf("error sorting percentiles: %w", err)
	}
	if sorted != nil {
		mean, _ := sorted.Mean()
		fmt.Fprintf(w, "axslog.latency_%s.average\t%f\t%d\n", keyPrefix, mean, now)
		for _, target := range percentileTargets {
			pValue, err := sorted.Percentile(target.Value)
			if err != nil {
				log.Printf("error getting percentile %s: %v\n", target.Name, err)
			} else {
				fmt.Fprintf(w, "axslog.latency_%s.%s\t%f\t%d\n", keyPrefix, target.Name, pValue, now)
			}
		}
	}
	return nil
}

// Display returns a string representation of the statistics, including percentiles and ratios, formatted for output.
func (s *Stats) Display(percentileTargets []PercentileTarget, keyPrefix string) string {
	var buf bytes.Buffer
	now := uint64(time.Now().Unix())

	err := displayPercentiles(&buf, s.percentiles, percentileTargets, keyPrefix, now)
	if err != nil {
		log.Printf("error displaying percentiles: %v\n", err)
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

// DisplayAll returns a string representation of the aggregated statistics from multiple Stats instances, including percentiles and ratios, formatted for output.
func DisplayAll(statsAll []*Stats, percentileTargets []PercentileTarget, keyPrefix string) string {
	var buf bytes.Buffer
	now := uint64(time.Now().Unix())

	allPercentiles := sampdo.New(sampdo.WithInitialCapacity(1024))
	c1xx := float64(0)
	c2xx := float64(0)
	c3xx := float64(0)
	c4xx := float64(0)
	c499 := float64(0)
	c5xx := float64(0)
	total := float64(0)
	allDurationNG := true
	for _, s := range statsAll {
		if s.percentiles != nil {
			if err := s.percentiles.AppendTo(allPercentiles); err != nil {
				log.Printf("error appending to all percentiles: %v\n", err)
			}
		}
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

	err := displayPercentiles(&buf, allPercentiles, percentileTargets, keyPrefix, now)
	if err != nil {
		log.Printf("error displaying all percentiles: %v\n", err)
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
