package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Normalizer func(deviceID, source string, body []byte) ([]Record, error)

type ccusageDaily struct {
	Daily []struct {
		Date        string  `json:"date"`
		TotalTokens int64   `json:"totalTokens"`
		TotalCost   float64 `json:"totalCost"`
	} `json:"daily"`
}

func normalizeCcusage(deviceID, source string, body []byte) ([]Record, error) {
	var c ccusageDaily
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("ccusage parse: %w", err)
	}
	out := make([]Record, 0, len(c.Daily))
	for _, d := range c.Daily {
		date, err := parseLooseDate(d.Date)
		if err != nil {
			return nil, fmt.Errorf("ccusage date %q: %w", d.Date, err)
		}
		out = append(out, Record{
			DeviceID:    deviceID,
			Source:      source,
			Date:        date,
			TotalTokens: d.TotalTokens,
			CostUSD:     d.TotalCost,
		})
	}
	return out, nil
}

type codexDaily struct {
	Daily []struct {
		Date        string  `json:"date"`
		TotalTokens int64   `json:"totalTokens"`
		CostUSD     float64 `json:"costUSD"`
	} `json:"daily"`
}

func normalizeCodex(deviceID, source string, body []byte) ([]Record, error) {
	var c codexDaily
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("codex parse: %w", err)
	}
	out := make([]Record, 0, len(c.Daily))
	for _, d := range c.Daily {
		date, err := parseLooseDate(d.Date)
		if err != nil {
			return nil, fmt.Errorf("codex date %q: %w", d.Date, err)
		}
		out = append(out, Record{
			DeviceID:    deviceID,
			Source:      source,
			Date:        date,
			TotalTokens: d.TotalTokens,
			CostUSD:     d.CostUSD,
		})
	}
	return out, nil
}

type standardDaily struct {
	Daily []struct {
		Date        string  `json:"date"`
		TotalTokens int64   `json:"total_tokens"`
		CostUSD     float64 `json:"cost_usd"`
	} `json:"daily"`
}

func normalizeStandard(deviceID, source string, body []byte) ([]Record, error) {
	var c standardDaily
	if err := json.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("standard parse: %w", err)
	}
	out := make([]Record, 0, len(c.Daily))
	for _, d := range c.Daily {
		date, err := parseLooseDate(d.Date)
		if err != nil {
			return nil, fmt.Errorf("standard date %q: %w", d.Date, err)
		}
		out = append(out, Record{
			DeviceID:    deviceID,
			Source:      source,
			Date:        date,
			TotalTokens: d.TotalTokens,
			CostUSD:     d.CostUSD,
		})
	}
	return out, nil
}

type hermesPayload struct {
	DailyByModel []struct {
		Date             string `json:"date"`
		Model            string `json:"model"`
		InputTokens      int64  `json:"input_tokens"`
		OutputTokens     int64  `json:"output_tokens"`
		CacheReadTokens  int64  `json:"cache_read_tokens"`
		CacheWriteTokens int64  `json:"cache_write_tokens"`
		ReasoningTokens  int64  `json:"reasoning_tokens"`
	} `json:"daily_by_model"`
}

func (s *Server) normalizeHermes(deviceID, source string, body []byte) ([]Record, error) {
	var p hermesPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("hermes parse: %w", err)
	}
	daily := map[string]*Record{}
	for _, row := range p.DailyByModel {
		date, err := parseLooseDate(row.Date)
		if err != nil {
			return nil, fmt.Errorf("hermes date %q: %w", row.Date, err)
		}
		rec, ok := daily[date]
		if !ok {
			rec = &Record{DeviceID: deviceID, Source: source, Date: date}
			daily[date] = rec
		}
		rec.TotalTokens += row.InputTokens + row.OutputTokens + row.CacheReadTokens + row.CacheWriteTokens + row.ReasoningTokens
		rec.CostUSD += s.Prices.Cost(row.Model, row.InputTokens, row.OutputTokens, row.CacheReadTokens, row.CacheWriteTokens, row.ReasoningTokens)
	}
	out := make([]Record, 0, len(daily))
	for _, r := range daily {
		out = append(out, *r)
	}
	return out, nil
}

func parseLooseDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02", "Jan 2, 2006", "Jan _2, 2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("unsupported date format")
}
