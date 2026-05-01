package main

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultHistoryDays = 30
	maxHistoryDays     = 365
)

type SourceDaily struct {
	TotalTokens int64   `json:"total_tokens"`
	CostUSD     float64 `json:"cost_usd"`
}

type DailyEntry struct {
	Date        string                 `json:"date"`
	TotalTokens int64                  `json:"total_tokens"`
	CostUSD     float64                `json:"cost_usd"`
	Sources     map[string]SourceDaily `json:"sources"`
}

type HistorySummary struct {
	TotalTokens      int64   `json:"total_tokens"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
	AvgTokensPerDay  int64   `json:"avg_tokens_per_day"`
	AvgCostUSDPerDay float64 `json:"avg_cost_usd_per_day"`
}

type History struct {
	Today   string         `json:"today"`
	From    string         `json:"from"`
	To      string         `json:"to"`
	Days    int            `json:"days"`
	Summary HistorySummary `json:"summary"`
	Daily   []DailyEntry   `json:"daily"`
}

func (s *Store) BuildHistory(today string, days int) (*History, error) {
	if days <= 0 {
		return nil, errors.New("days must be positive")
	}
	todayDate, err := time.Parse("2006-01-02", today)
	if err != nil {
		return nil, err
	}
	fromDate := todayDate.AddDate(0, 0, -(days - 1))
	from := fromDate.Format("2006-01-02")
	to := todayDate.Format("2006-01-02")

	daily := make([]DailyEntry, days)
	byDate := make(map[string]*DailyEntry, days)
	for i := 0; i < days; i++ {
		date := fromDate.AddDate(0, 0, i).Format("2006-01-02")
		daily[i] = DailyEntry{Date: date, Sources: map[string]SourceDaily{}}
		byDate[date] = &daily[i]
	}

	rows, err := s.db.Query(`
		SELECT date, source, SUM(total_tokens), SUM(cost_usd)
		FROM usage
		WHERE date >= ? AND date <= ?
		GROUP BY date, source
		ORDER BY date ASC, source ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hist := &History{Today: to, From: from, To: to, Days: days, Daily: daily}
	for rows.Next() {
		var date, source string
		var tokens int64
		var cost float64
		if err := rows.Scan(&date, &source, &tokens, &cost); err != nil {
			return nil, err
		}
		entry := byDate[date]
		if entry == nil {
			continue
		}
		entry.TotalTokens += tokens
		entry.CostUSD += cost
		entry.Sources[source] = SourceDaily{TotalTokens: tokens, CostUSD: cost}
		hist.Summary.TotalTokens += tokens
		hist.Summary.TotalCostUSD += cost
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hist.Summary.AvgTokensPerDay = hist.Summary.TotalTokens / int64(days)
	hist.Summary.AvgCostUSDPerDay = hist.Summary.TotalCostUSD / float64(days)
	return hist, nil
}

func parseHistoryDays(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("days")
	if raw == "" {
		return defaultHistoryDays, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if days <= 0 || days > maxHistoryDays {
		return 0, errors.New("days must be between 1 and 365")
	}
	return days, nil
}

func (s *Server) historyToday(r *http.Request) string {
	if today := r.URL.Query().Get("today"); today != "" {
		return today
	}
	return time.Now().In(mustLoc(s.Timezone)).Format("2006-01-02")
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	days, err := parseHistoryDays(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	today := s.historyToday(r)
	if _, err := time.Parse("2006-01-02", today); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hist, err := s.Store.BuildHistory(today, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, hist)
}
