package main

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHistoryDays = 30
	maxHistoryDays     = 365
)

type Cell struct {
	TotalTokens int64   `json:"total_tokens"`
	CostUSD     float64 `json:"cost_usd"`
}

type DailyEntry struct {
	Date        string                     `json:"date"`
	TotalTokens int64                      `json:"total_tokens"`
	CostUSD     float64                    `json:"cost_usd"`
	Sources     map[string]Cell            `json:"sources"`
	Devices     map[string]Cell            `json:"devices"`
	Breakdown   map[string]map[string]Cell `json:"breakdown"`
}

type HistorySummary struct {
	TotalTokens      int64   `json:"total_tokens"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
	AvgTokensPerDay  int64   `json:"avg_tokens_per_day"`
	AvgCostUSDPerDay float64 `json:"avg_cost_usd_per_day"`
}

type HistoryFilters struct {
	Devices []string `json:"devices"`
	Sources []string `json:"sources"`
}

type History struct {
	Today   string         `json:"today"`
	From    string         `json:"from"`
	To      string         `json:"to"`
	Days    int            `json:"days"`
	Devices []string       `json:"devices"`
	Sources []string       `json:"sources"`
	Filters HistoryFilters `json:"filters"`
	Summary HistorySummary `json:"summary"`
	Daily   []DailyEntry   `json:"daily"`
}

func (s *Store) BuildHistory(today string, days int, filter HistoryFilters) (*History, error) {
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

	devices, sources, err := s.distinctDimensions(from, to)
	if err != nil {
		return nil, err
	}

	daily := make([]DailyEntry, days)
	byDate := make(map[string]*DailyEntry, days)
	for i := 0; i < days; i++ {
		date := fromDate.AddDate(0, 0, i).Format("2006-01-02")
		daily[i] = DailyEntry{
			Date:      date,
			Sources:   map[string]Cell{},
			Devices:   map[string]Cell{},
			Breakdown: map[string]map[string]Cell{},
		}
		byDate[date] = &daily[i]
	}

	where := []string{"date >= ?", "date <= ?"}
	args := []any{from, to}
	if len(filter.Devices) > 0 {
		where = append(where, "device_id IN ("+placeholders(len(filter.Devices))+")")
		for _, d := range filter.Devices {
			args = append(args, d)
		}
	}
	if len(filter.Sources) > 0 {
		where = append(where, "source IN ("+placeholders(len(filter.Sources))+")")
		for _, s := range filter.Sources {
			args = append(args, s)
		}
	}
	q := `SELECT date, device_id, source, SUM(total_tokens), SUM(cost_usd) FROM usage WHERE ` +
		strings.Join(where, " AND ") +
		` GROUP BY date, device_id, source ORDER BY date ASC, device_id ASC, source ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hist := &History{
		Today:   to,
		From:    from,
		To:      to,
		Days:    days,
		Devices: devices,
		Sources: sources,
		Filters: HistoryFilters{
			Devices: append([]string(nil), filter.Devices...),
			Sources: append([]string(nil), filter.Sources...),
		},
		Daily: daily,
	}
	for rows.Next() {
		var date, device, source string
		var tokens int64
		var cost float64
		if err := rows.Scan(&date, &device, &source, &tokens, &cost); err != nil {
			return nil, err
		}
		entry := byDate[date]
		if entry == nil {
			continue
		}
		entry.TotalTokens += tokens
		entry.CostUSD += cost

		src := entry.Sources[source]
		src.TotalTokens += tokens
		src.CostUSD += cost
		entry.Sources[source] = src

		dev := entry.Devices[device]
		dev.TotalTokens += tokens
		dev.CostUSD += cost
		entry.Devices[device] = dev

		if entry.Breakdown[device] == nil {
			entry.Breakdown[device] = map[string]Cell{}
		}
		entry.Breakdown[device][source] = Cell{TotalTokens: tokens, CostUSD: cost}

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

func (s *Store) distinctDimensions(from, to string) ([]string, []string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT device_id, source FROM usage WHERE date >= ? AND date <= ?`,
		from, to)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	devSet := map[string]struct{}{}
	srcSet := map[string]struct{}{}
	for rows.Next() {
		var d, s string
		if err := rows.Scan(&d, &s); err != nil {
			return nil, nil, err
		}
		devSet[d] = struct{}{}
		srcSet[s] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	devices := make([]string, 0, len(devSet))
	for d := range devSet {
		devices = append(devices, d)
	}
	sources := make([]string, 0, len(srcSet))
	for s := range srcSet {
		sources = append(sources, s)
	}
	sort.Strings(devices)
	sort.Strings(sources)
	return devices, sources, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
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

func parseFilterValues(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
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
	filter := HistoryFilters{
		Devices: parseFilterValues(r.URL.Query().Get("devices")),
		Sources: parseFilterValues(r.URL.Query().Get("sources")),
	}
	hist, err := s.Store.BuildHistory(today, days, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, hist)
}
