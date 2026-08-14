// Package stream handles Git and text stream processing for nopii.
package stream

import "github.com/iilei/nopii/internal/config"

// ClampTimestamp floors ts to the nearest granularity_seconds boundary.
// It returns ts unchanged when clamping is disabled or granularity is invalid.
func ClampTimestamp(ts int64, cfg config.DateClampConfig) int64 {
	if !cfg.Enabled || cfg.GranularitySeconds < 1 {
		return ts
	}
	g := int64(cfg.GranularitySeconds)
	return (ts / g) * g
}
