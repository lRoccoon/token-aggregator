package timezonebug

import "time"

// Event is a token usage event emitted by a device. Occurred is always a UTC timestamp.
type Event struct {
	ID          string
	DeviceID    string
	Source      string
	Occurred    time.Time
	TotalTokens int64
	CostUSD     float64
}

// DailyUsage is the aggregated usage for one business day.
type DailyUsage struct {
	Date        string
	TotalTokens int64
	CostUSD     float64
}

// CountByBusinessDay aggregates UTC events into business-day buckets.
//
// Business requirement: buckets are natural days in Asia/Shanghai, not UTC days.
func CountByBusinessDay(events []Event) []DailyUsage {
	byDay := make(map[string]DailyUsage)
	for _, event := range events {
		// BUG: this uses the timestamp's current location, which is UTC for ingested events.
		// Events around 00:00 in Asia/Shanghai are assigned to the previous UTC date.
		day := event.Occurred.Format("2006-01-02")
		usage := byDay[day]
		usage.Date = day
		usage.TotalTokens += event.TotalTokens
		usage.CostUSD += event.CostUSD
		byDay[day] = usage
	}

	out := make([]DailyUsage, 0, len(byDay))
	for _, usage := range byDay {
		out = append(out, usage)
	}
	return out
}
