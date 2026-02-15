package benchmark

import (
	"sort"
	"time"
)

// AvgDuration returns the average of the given durations.
// Returns 0 for an empty slice.
func AvgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}

	return sum / time.Duration(len(durations))
}

// MinDuration returns the smallest duration in the slice.
// Returns 0 for an empty slice.
func MinDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	result := durations[0]
	for _, d := range durations[1:] {
		if d < result {
			result = d
		}
	}

	return result
}

// MaxDuration returns the largest duration in the slice.
// Returns 0 for an empty slice.
func MaxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	result := durations[0]
	for _, d := range durations[1:] {
		if d > result {
			result = d
		}
	}

	return result
}

// Percentile returns the p-th percentile (0.0–1.0) of the given durations.
// Returns 0 for an empty slice.
func Percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
