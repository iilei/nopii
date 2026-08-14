package stream_test

import (
	"testing"

	"github.com/iilei/nopii/internal/config"
	streampkg "github.com/iilei/nopii/internal/stream"
)

func TestClampTimestamp_Disabled(t *testing.T) {
	cfg := config.DateClampConfig{Enabled: false, GranularitySeconds: 86400}
	ts := int64(1_700_000_999)
	if got := streampkg.ClampTimestamp(ts, cfg); got != ts {
		t.Fatalf("expected passthrough %d, got %d", ts, got)
	}
}

func TestClampTimestamp_Floor(t *testing.T) {
	cfg := config.DateClampConfig{Enabled: true, GranularitySeconds: 86400}
	// 2023-11-14 10:09:59 UTC  →  floor to 2023-11-14 00:00:00 UTC
	ts := int64(1_700_000_999)
	want := int64(1_699_920_000) // 1700000999 / 86400 * 86400
	if got := streampkg.ClampTimestamp(ts, cfg); got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
}

func TestClampTimestamp_Deterministic(t *testing.T) {
	cfg := config.DateClampConfig{Enabled: true, GranularitySeconds: 3600}
	ts := int64(1_700_012_345)
	first := streampkg.ClampTimestamp(ts, cfg)
	for range 10 {
		if got := streampkg.ClampTimestamp(ts, cfg); got != first {
			t.Fatal("clamp is not deterministic")
		}
	}
}

func TestClampTimestamp_Boundary(t *testing.T) {
	cfg := config.DateClampConfig{Enabled: true, GranularitySeconds: 86400}
	ts := int64(86400) // exactly on a boundary
	if got := streampkg.ClampTimestamp(ts, cfg); got != ts {
		t.Fatalf("expected %d, got %d", ts, got)
	}
}

func TestClampTimestamp_InvalidGranularity(t *testing.T) {
	cfg := config.DateClampConfig{Enabled: true, GranularitySeconds: 0}
	ts := int64(1_700_000_999)
	if got := streampkg.ClampTimestamp(ts, cfg); got != ts {
		t.Fatalf("expected passthrough on invalid granularity, got %d", got)
	}
}
