package timezonebug

import (
	"testing"
	"time"
)

func TestCountByBusinessDay_NormalCase(t *testing.T) {
	events := []Event{
		{ID: "1", DeviceID: "macbook", Source: "hermes", Occurred: mustParseRFC3339(t, "2026-05-17T01:00:00Z"), TotalTokens: 100, CostUSD: 0.10},
		{ID: "2", DeviceID: "macbook", Source: "codex", Occurred: mustParseRFC3339(t, "2026-05-17T02:00:00Z"), TotalTokens: 200, CostUSD: 0.20},
	}

	byDate := indexByDate(CountByBusinessDay(events))
	got := byDate["2026-05-17"]

	if got.TotalTokens != 300 {
		t.Fatalf("2026-05-17 total tokens = %d, want 300; full result: %#v", got.TotalTokens, byDate)
	}
}

func TestCountByBusinessDay_UsesShanghaiBusinessDay(t *testing.T) {
	events := []Event{
		// 2026-05-16 16:30 UTC is 2026-05-17 00:30 in Asia/Shanghai.
		{ID: "1", DeviceID: "macbook", Source: "hermes", Occurred: mustParseRFC3339(t, "2026-05-16T16:30:00Z"), TotalTokens: 100, CostUSD: 0.10},
		{ID: "2", DeviceID: "macbook", Source: "codex", Occurred: mustParseRFC3339(t, "2026-05-17T01:00:00Z"), TotalTokens: 200, CostUSD: 0.20},
	}

	byDate := indexByDate(CountByBusinessDay(events))
	got := byDate["2026-05-17"]

	if got.TotalTokens != 300 {
		t.Fatalf("2026-05-17 Shanghai business-day tokens = %d, want 300; full result: %#v", got.TotalTokens, byDate)
	}
	if _, ok := byDate["2026-05-16"]; ok {
		t.Fatalf("UTC day 2026-05-16 should not be present when aggregating by Shanghai business day; full result: %#v", byDate)
	}
}

func indexByDate(usages []DailyUsage) map[string]DailyUsage {
	out := make(map[string]DailyUsage, len(usages))
	for _, usage := range usages {
		out[usage.Date] = usage
	}
	return out
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
