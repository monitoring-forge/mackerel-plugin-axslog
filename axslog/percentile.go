package axslog

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type PercentileTarget struct {
	Name  string  // e.g., "50_percentile", "99_9_percentile"
	Value float64 // e.g., 50.0, 90.0
}

func PercentileTargetFromString(p string) (PercentileTarget, error) {
	f, err := strconv.ParseFloat(p, 64)
	if err != nil || math.IsNaN(f) || f < 0 || f > 100 {
		return PercentileTarget{}, fmt.Errorf("invalid percentile value: %s", p)
	}
	name := strings.ReplaceAll(strconv.FormatFloat(f, 'f', -1, 64), ".", "_") + "_percentile"
	return PercentileTarget{
		Name:  name,
		Value: f,
	}, nil
}

func PercentileTargetsFromString(s string) ([]PercentileTarget, error) {
	if s == "" {
		return nil, fmt.Errorf("percentiles string must not be empty")
	}
	var targets []PercentileTarget
	parts := strings.Split(s, ",")
	seen := make(map[string]struct{})
	for _, p := range parts {
		target, err := PercentileTargetFromString(p)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[target.Name]; !exists {
			targets = append(targets, target)
			seen[target.Name] = struct{}{}
		}
	}
	return targets, nil
}
