package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const defaultSlotAPIURL = "https://l.garyyang.work/api/slot/update"

const defaultSlotTemplate = `已经消耗词元：今日 {{tokens .TodayTokens}} / 总计 {{tokens .TotalTokens}}，白赚 {{money .TotalCost}}`

type SlotConfig struct {
	SlotID       string
	Credential   string
	APIURL       string
	Interval     time.Duration
	TemplatePath string
	Timezone     string
}

type SlotPusher struct {
	cfg         SlotConfig
	store       *Store
	client      *http.Client
	funcs       template.FuncMap
	lastSummary string
}

func NewSlotPusher(cfg SlotConfig, store *Store) *SlotPusher {
	return &SlotPusher{
		cfg:    cfg,
		store:  store,
		client: &http.Client{Timeout: 15 * time.Second},
		funcs: template.FuncMap{
			"tokens":   humanTokens,
			"millions": humanTokens, // 向后兼容旧模板
			"money":    func(f float64) string { return fmt.Sprintf("$%.2f", f) },
		},
	}
}

// humanTokens 按进制自动选择单位输出 token 计数：< 1K 显示整数，
// 其余使用 K / M / B / T。小于 10 的量保留一位小数，其余取整。
func humanTokens(n int64) string {
	if n < 0 {
		return "-" + humanTokens(-n)
	}
	const (
		k = 1_000
		m = 1_000_000
		b = 1_000_000_000
		t = 1_000_000_000_000
	)
	switch {
	case n >= t:
		return formatTokenUnit(float64(n)/float64(t), "T")
	case n >= b:
		return formatTokenUnit(float64(n)/float64(b), "B")
	case n >= m:
		return formatTokenUnit(float64(n)/float64(m), "M")
	case n >= k:
		return formatTokenUnit(float64(n)/float64(k), "K")
	default:
		return strconv.FormatInt(n, 10)
	}
}

func formatTokenUnit(v float64, unit string) string {
	if v < 10 {
		s := strconv.FormatFloat(v, 'f', 1, 64)
		s = strings.TrimSuffix(s, ".0")
		return s + unit
	}
	return strconv.FormatFloat(v, 'f', 0, 64) + unit
}

func (p *SlotPusher) Run(ctx context.Context) {
	log.Printf("slot pusher: enabled slot=%s interval=%s template=%s", p.cfg.SlotID, p.cfg.Interval, p.cfg.TemplatePath)
	if err := p.pushOnce(ctx); err != nil {
		log.Printf("slot pusher: initial push failed: %v", err)
	}
	ticker := time.NewTicker(p.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.pushOnce(ctx); err != nil {
				log.Printf("slot pusher: push failed: %v", err)
			}
		}
	}
}

func (p *SlotPusher) pushOnce(ctx context.Context) error {
	today := time.Now().In(mustLoc(p.cfg.Timezone)).Format("2006-01-02")
	rep, err := p.store.BuildReport(today)
	if err != nil {
		return fmt.Errorf("build report: %w", err)
	}
	summary, err := p.renderTemplate(rep)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}
	if summary == p.lastSummary {
		log.Printf("slot pusher: skip (summary unchanged)")
		return nil
	}
	if err := p.callSlotAPI(ctx, summary); err != nil {
		return err
	}
	p.lastSummary = summary
	return nil
}

func (p *SlotPusher) renderTemplate(rep *Report) (string, error) {
	tmplText := defaultSlotTemplate
	if p.cfg.TemplatePath != "" {
		data, err := os.ReadFile(p.cfg.TemplatePath)
		if err == nil {
			text := strings.TrimRight(string(data), "\r\n")
			if text != "" {
				tmplText = text
			}
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("read template: %w", err)
		}
	}
	tmpl, err := template.New("slot").Funcs(p.funcs).Parse(tmplText)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, rep); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

func (p *SlotPusher) callSlotAPI(ctx context.Context, summary string) error {
	payload, err := json.Marshal(map[string]string{
		"slotId": p.cfg.SlotID,
		"value":  summary,
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.APIURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.Credential)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("call slot api: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("slot api status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	log.Printf("slot pusher: pushed summary=%q response=%s", summary, strings.TrimSpace(string(body)))
	return nil
}
