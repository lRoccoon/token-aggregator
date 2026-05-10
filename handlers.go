package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed scripts/install.sh
var installScriptRaw []byte

//go:embed scripts/collector.sh
var collectorScriptRaw []byte

//go:embed docs/home.md
var homeMarkdownRaw []byte

type Server struct {
	Store       *Store
	Token       string
	PublicURL   string
	Timezone    string
	MaxBodySize int64
	Prices      *PriceBook
	normalizers map[string]Normalizer
}

func (s *Server) routes() *http.ServeMux {
	s.normalizers = map[string]Normalizer{
		"ccusage":  normalizeCcusage,
		"codex":    normalizeCodex,
		"standard": normalizeStandard,
		"hermes":   s.normalizeHermes,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/ingest", s.handleIngest)
	mux.HandleFunc("/report", s.handleReport)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/dashboard", s.handleDashboard)
	mux.HandleFunc("/unknown-models", s.handleUnknownModels)
	mux.HandleFunc("/install.sh", s.handleInstall)
	mux.HandleFunc("/collector.sh", s.handleCollector)
	mux.HandleFunc("/", s.handleRoot)
	return mux
}

func (s *Server) handleUnknownModels(w http.ResponseWriter, r *http.Request) {
	if !s.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, s.Prices.UnknownModels())
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	md := s.renderScript(homeMarkdownRaw, r)
	if isBrowser(r.Header.Get("User-Agent")) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(renderHomeHTML(md))
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(md)
}

// isBrowser identifies real browsers by the "Mozilla/" UA prefix. curl, wget,
// httpie, Go's default client, etc. don't send that.
func isBrowser(ua string) bool {
	return strings.HasPrefix(ua, "Mozilla/")
}

func renderHomeHTML(markdown []byte) []byte {
	payload, _ := json.Marshal(string(markdown))
	var buf bytes.Buffer
	buf.WriteString(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>token-aggregator</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
body{max-width:820px;margin:2em auto;padding:0 1em;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Hiragino Sans GB","Microsoft YaHei",sans-serif;line-height:1.6;color:#222;background:#fff}
h1,h2,h3{line-height:1.25;margin-top:1.4em}
h1{border-bottom:1px solid #eee;padding-bottom:.3em}
h2{border-bottom:1px solid #eee;padding-bottom:.25em;font-size:1.35em}
code{background:#f3f3f5;padding:.1em .35em;border-radius:3px;font-size:.92em;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
pre{background:#f6f8fa;padding:.9em 1em;border-radius:6px;overflow-x:auto}
pre code{background:transparent;padding:0;font-size:.88em}
a{color:#0969da;text-decoration:none}
a:hover{text-decoration:underline}
table{border-collapse:collapse;margin:.6em 0}
th,td{border:1px solid #d0d7de;padding:.4em .8em;text-align:left}
th{background:#f6f8fa}
hr{border:0;border-top:1px solid #eee}
</style>
</head>
<body>
<article id="app">Loading…</article>
<script src="https://cdn.jsdelivr.net/npm/marked@12/marked.min.js"></script>
<script>
const src = `)
	buf.Write(payload)
	buf.WriteString(`;
document.getElementById('app').innerHTML = marked.parse(src);
</script>
</body>
</html>`)
	return buf.Bytes()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	deviceID := strings.TrimSpace(r.Header.Get("X-Device-Id"))
	source := strings.TrimSpace(r.Header.Get("X-Source"))
	format := strings.TrimSpace(r.Header.Get("X-Format"))
	if deviceID == "" || source == "" {
		http.Error(w, "missing X-Device-Id or X-Source", http.StatusBadRequest)
		return
	}
	if format == "" {
		format = "standard"
	}
	normalize, ok := s.normalizers[format]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown X-Format %q", format), http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, s.MaxBodySize))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	records, err := normalize(deviceID, source, body)
	if err != nil {
		log.Printf("ingest normalize device=%s source=%s format=%s err=%v", deviceID, source, format, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Store.Upsert(records); err != nil {
		log.Printf("ingest store device=%s source=%s err=%v", deviceID, source, err)
		http.Error(w, "store: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("ingest ok device=%s source=%s format=%s rows=%d", deviceID, source, format, len(records))
	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"accepted": len(records),
	})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if !s.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	today := time.Now().In(mustLoc(s.Timezone)).Format("2006-01-02")
	if t := r.URL.Query().Get("today"); t != "" {
		today = t
	}
	rep, err := s.Store.BuildReport(today)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	_, _ = w.Write(s.renderScript(installScriptRaw, r))
}

func (s *Server) handleCollector(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	_, _ = w.Write(s.renderScript(collectorScriptRaw, r))
}

func (s *Server) renderScript(script []byte, r *http.Request) []byte {
	serverURL := s.PublicURL
	if serverURL == "" {
		scheme := "http"
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		serverURL = fmt.Sprintf("%s://%s", scheme, r.Host)
	}
	return bytes.ReplaceAll(script, []byte("__SERVER_URL__"), []byte(serverURL))
}

func (s *Server) checkAuth(r *http.Request) bool {
	if s.Token == "" {
		return true
	}
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return got == s.Token
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func mustLoc(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}
