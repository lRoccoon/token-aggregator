package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestBuildHistoryEmpty(t *testing.T) {
	store := openTestStore(t)
	hist, err := store.BuildHistory("2026-05-02", 7)
	if err != nil {
		t.Fatalf("BuildHistory: %v", err)
	}
	if hist.Today != "2026-05-02" {
		t.Errorf("Today = %q, want 2026-05-02", hist.Today)
	}
	if hist.Days != 7 {
		t.Errorf("Days = %d, want 7", hist.Days)
	}
	if hist.From != "2026-04-26" {
		t.Errorf("From = %q, want 2026-04-26", hist.From)
	}
	if hist.To != "2026-05-02" {
		t.Errorf("To = %q, want 2026-05-02", hist.To)
	}
	if len(hist.Daily) != 7 {
		t.Fatalf("len(Daily) = %d, want 7", len(hist.Daily))
	}
	for i, d := range hist.Daily {
		if d.TotalTokens != 0 || d.CostUSD != 0 {
			t.Errorf("Daily[%d] = %+v, want zero", i, d)
		}
		if d.Sources == nil {
			t.Errorf("Daily[%d].Sources is nil; want empty map", i)
		}
	}
	if hist.Daily[0].Date != "2026-04-26" || hist.Daily[6].Date != "2026-05-02" {
		t.Errorf("Daily date range = %q..%q, want 2026-04-26..2026-05-02",
			hist.Daily[0].Date, hist.Daily[6].Date)
	}
	if hist.Summary.TotalTokens != 0 || hist.Summary.TotalCostUSD != 0 {
		t.Errorf("Summary should be zero, got %+v", hist.Summary)
	}
}

func TestBuildHistoryAggregatesAndWindowing(t *testing.T) {
	store := openTestStore(t)
	if err := store.Upsert([]Record{
		{DeviceID: "laptop", Source: "claude", Date: "2026-05-02", TotalTokens: 1000, CostUSD: 0.10},
		{DeviceID: "laptop", Source: "codex", Date: "2026-05-02", TotalTokens: 500, CostUSD: 0.05},
		{DeviceID: "vps", Source: "claude", Date: "2026-05-02", TotalTokens: 200, CostUSD: 0.02},
		{DeviceID: "laptop", Source: "claude", Date: "2026-04-30", TotalTokens: 300, CostUSD: 0.03},
		{DeviceID: "laptop", Source: "claude", Date: "2026-04-25", TotalTokens: 9999, CostUSD: 9.99}, // outside 7d window
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	hist, err := store.BuildHistory("2026-05-02", 7)
	if err != nil {
		t.Fatalf("BuildHistory: %v", err)
	}
	if len(hist.Daily) != 7 {
		t.Fatalf("len(Daily) = %d, want 7", len(hist.Daily))
	}

	byDate := map[string]DailyEntry{}
	for _, d := range hist.Daily {
		byDate[d.Date] = d
	}

	d502 := byDate["2026-05-02"]
	if d502.TotalTokens != 1700 {
		t.Errorf("2026-05-02 tokens = %d, want 1700", d502.TotalTokens)
	}
	if !floatEq(d502.CostUSD, 0.17) {
		t.Errorf("2026-05-02 cost = %f, want 0.17", d502.CostUSD)
	}
	if d502.Sources["claude"].TotalTokens != 1200 {
		t.Errorf("2026-05-02 claude tokens = %d, want 1200", d502.Sources["claude"].TotalTokens)
	}
	if !floatEq(d502.Sources["claude"].CostUSD, 0.12) {
		t.Errorf("2026-05-02 claude cost = %f, want 0.12", d502.Sources["claude"].CostUSD)
	}
	if d502.Sources["codex"].TotalTokens != 500 {
		t.Errorf("2026-05-02 codex tokens = %d, want 500", d502.Sources["codex"].TotalTokens)
	}

	d430 := byDate["2026-04-30"]
	if d430.TotalTokens != 300 || !floatEq(d430.CostUSD, 0.03) {
		t.Errorf("2026-04-30 = %+v, want 300 tokens / 0.03 cost", d430)
	}

	d426 := byDate["2026-04-26"]
	if d426.TotalTokens != 0 || d426.CostUSD != 0 {
		t.Errorf("2026-04-26 should be zero, got %+v", d426)
	}
	if _, ok := byDate["2026-04-25"]; ok {
		t.Errorf("2026-04-25 must not be in window")
	}

	if hist.Summary.TotalTokens != 2000 {
		t.Errorf("Summary total tokens = %d, want 2000", hist.Summary.TotalTokens)
	}
	if !floatEq(hist.Summary.TotalCostUSD, 0.20) {
		t.Errorf("Summary total cost = %f, want 0.20", hist.Summary.TotalCostUSD)
	}
	// 2000 / 7 ≈ 285
	if hist.Summary.AvgTokensPerDay != 285 {
		t.Errorf("Summary avg tokens/day = %d, want 285", hist.Summary.AvgTokensPerDay)
	}
	// 0.20 / 7 ≈ 0.02857
	if !floatEq(hist.Summary.AvgCostUSDPerDay, 0.20/7.0) {
		t.Errorf("Summary avg cost/day = %f, want %f", hist.Summary.AvgCostUSDPerDay, 0.20/7.0)
	}
}

func TestBuildHistoryDaysOne(t *testing.T) {
	store := openTestStore(t)
	if err := store.Upsert([]Record{
		{DeviceID: "laptop", Source: "claude", Date: "2026-05-02", TotalTokens: 100, CostUSD: 0.01},
		{DeviceID: "laptop", Source: "claude", Date: "2026-05-01", TotalTokens: 999, CostUSD: 9.99},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	hist, err := store.BuildHistory("2026-05-02", 1)
	if err != nil {
		t.Fatalf("BuildHistory: %v", err)
	}
	if len(hist.Daily) != 1 || hist.Daily[0].Date != "2026-05-02" || hist.Daily[0].TotalTokens != 100 {
		t.Fatalf("Daily=%+v, want single entry 2026-05-02/100", hist.Daily)
	}
	if hist.From != "2026-05-02" || hist.To != "2026-05-02" {
		t.Errorf("From/To = %s/%s, want 2026-05-02/2026-05-02", hist.From, hist.To)
	}
}

func TestBuildHistoryRejectsNonPositiveDays(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.BuildHistory("2026-05-02", 0); err == nil {
		t.Errorf("BuildHistory(days=0) should error")
	}
	if _, err := store.BuildHistory("2026-05-02", -3); err == nil {
		t.Errorf("BuildHistory(days=-3) should error")
	}
}

func TestBuildHistoryRejectsBadDate(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.BuildHistory("not-a-date", 7); err == nil {
		t.Errorf("BuildHistory should reject malformed today")
	}
}

func newTestServer(t *testing.T, token string) *Server {
	t.Helper()
	store := openTestStore(t)
	return &Server{
		Store:       store,
		Token:       token,
		Timezone:    "UTC",
		MaxBodySize: 1 << 20,
		Prices:      NewPriceBook(t.TempDir()),
	}
}

func TestHandleHistoryRequiresAuth(t *testing.T) {
	srv := newTestServer(t, "secret")
	mux := srv.routes()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/history?today=2026-05-02&days=7", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestHandleHistoryReturnsJSON(t *testing.T) {
	srv := newTestServer(t, "")
	if err := srv.Store.Upsert([]Record{
		{DeviceID: "laptop", Source: "claude", Date: "2026-05-02", TotalTokens: 1000, CostUSD: 0.10},
		{DeviceID: "laptop", Source: "claude", Date: "2026-04-30", TotalTokens: 300, CostUSD: 0.03},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	mux := srv.routes()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/history?today=2026-05-02&days=7", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var got History
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Days != 7 || got.Today != "2026-05-02" || got.From != "2026-04-26" {
		t.Errorf("got = %+v, want days=7 today=2026-05-02 from=2026-04-26", got)
	}
	if len(got.Daily) != 7 {
		t.Errorf("len(Daily) = %d, want 7", len(got.Daily))
	}
	if got.Summary.TotalTokens != 1300 {
		t.Errorf("Summary.TotalTokens = %d, want 1300", got.Summary.TotalTokens)
	}
}

func TestHandleHistoryDefaultsDaysTo30(t *testing.T) {
	srv := newTestServer(t, "")
	mux := srv.routes()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/history?today=2026-05-02", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	var got History
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Days != 30 {
		t.Errorf("Days = %d, want 30", got.Days)
	}
	if len(got.Daily) != 30 {
		t.Errorf("len(Daily) = %d, want 30", len(got.Daily))
	}
}

func TestHandleHistoryRejectsInvalidDays(t *testing.T) {
	srv := newTestServer(t, "")
	mux := srv.routes()
	for _, q := range []string{"days=0", "days=-1", "days=abc", "days=4000"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/history?today=2026-05-02&"+q, nil)
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("query %q: status = %d, want 400", q, rr.Code)
		}
	}
}

func TestHandleDashboardServesHTML(t *testing.T) {
	srv := newTestServer(t, "")
	mux := srv.routes()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct == "" || ct[:9] != "text/html" {
		t.Errorf("Content-Type = %q, want text/html...", ct)
	}
	body := rr.Body.String()
	for _, marker := range []string{"<html", "/history", "Chart"} {
		if !strings.Contains(body, marker) {
			t.Errorf("dashboard body missing marker %q", marker)
		}
	}
}

func TestDashboardEscapesDynamicValues(t *testing.T) {
	srv := newTestServer(t, "")
	mux := srv.routes()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.String()
	for _, marker := range []string{"const esc =", "esc(name)", "esc(Object.keys", "sessionStorage", "history.replaceState"} {
		if !strings.Contains(body, marker) {
			t.Errorf("dashboard body missing XSS/auth hardening marker %q", marker)
		}
	}
	if strings.Contains(body, "localStorage.setItem('tokenAggregatorToken'") {
		t.Errorf("dashboard must not persist bearer token to localStorage")
	}
}

func floatEq(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}
