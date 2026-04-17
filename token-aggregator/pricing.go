package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const litellmURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

type ModelPrice struct {
	InputCostPerToken           float64 `json:"input_cost_per_token"`
	OutputCostPerToken          float64 `json:"output_cost_per_token"`
	CacheReadInputTokenCost     float64 `json:"cache_read_input_token_cost"`
	CacheCreationInputTokenCost float64 `json:"cache_creation_input_token_cost"`
}

type PriceBook struct {
	cacheDir     string
	ttl          time.Duration
	mu           sync.RWMutex
	prices       map[string]ModelPrice
	aliases      map[string]ModelPrice
	overrides    map[string]ModelPrice
	loadedAt     time.Time
	unknownModel map[string]int
}

func NewPriceBook(cacheDir string) *PriceBook {
	return &PriceBook{
		cacheDir:     cacheDir,
		ttl:          24 * time.Hour,
		prices:       map[string]ModelPrice{},
		aliases:      map[string]ModelPrice{},
		overrides:    map[string]ModelPrice{},
		unknownModel: map[string]int{},
	}
}

func (p *PriceBook) litellmCachePath() string  { return filepath.Join(p.cacheDir, "litellm_prices.json") }
func (p *PriceBook) overridesPath() string     { return filepath.Join(p.cacheDir, "price_overrides.json") }

func (p *PriceBook) ensureFresh() {
	p.mu.RLock()
	fresh := !p.loadedAt.IsZero() && time.Since(p.loadedAt) < p.ttl
	p.mu.RUnlock()
	if fresh {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.loadedAt.IsZero() && time.Since(p.loadedAt) < p.ttl {
		return
	}
	if err := p.loadLitellm(); err != nil {
		log.Printf("pricing: %v (cost fallback to 0 for unknown models)", err)
	}
	if err := p.loadOverrides(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("pricing overrides: %v", err)
	}
	p.loadedAt = time.Now()
}

func (p *PriceBook) loadLitellm() error {
	path := p.litellmCachePath()
	var data []byte
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < p.ttl {
		data, err = os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return p.parseLitellm(data)
		}
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(litellmURL)
	if err != nil {
		if fallback, rerr := os.ReadFile(path); rerr == nil && len(fallback) > 0 {
			_ = p.parseLitellm(fallback)
			return fmt.Errorf("litellm fetch failed, using stale cache: %w", err)
		}
		return fmt.Errorf("litellm fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("litellm fetch status %d", resp.StatusCode)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("litellm read: %w", err)
	}
	if err := os.MkdirAll(p.cacheDir, 0755); err == nil {
		_ = os.WriteFile(path, data, 0644)
	}
	return p.parseLitellm(data)
}

func (p *PriceBook) parseLitellm(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("litellm parse: %w", err)
	}
	out := make(map[string]ModelPrice, len(raw))
	for k, v := range raw {
		if k == "sample_spec" {
			continue
		}
		var mp ModelPrice
		if err := json.Unmarshal(v, &mp); err != nil {
			continue
		}
		out[k] = mp
	}
	p.prices = out
	p.aliases = buildAliases(out)
	return nil
}

// buildAliases maps a "core" model name (provider/version stripped) to the
// price of the shortest litellm key that reduces to it. Lets us match names
// like "claude-3-5-sonnet-20241022" against litellm keys such as
// "anthropic.claude-3-5-sonnet-20241022-v2:0".
func buildAliases(prices map[string]ModelPrice) map[string]ModelPrice {
	keys := make([]string, 0, len(prices))
	for k := range prices {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) < len(keys[j]) })
	aliases := make(map[string]ModelPrice, len(keys))
	for _, k := range keys {
		core := coreName(k)
		if core == "" || core == k {
			continue
		}
		if _, exists := prices[core]; exists {
			continue
		}
		if _, exists := aliases[core]; !exists {
			aliases[core] = prices[k]
		}
	}
	return aliases
}

// coreName reduces a litellm key to its bare model name by stripping a leading
// "provider/..." path and a leading "provider." prefix (only when the prefix
// has no '-', which distinguishes "anthropic." from valid names like "gpt-5.4"),
// then any trailing "-v<digits>(:<digits>)?" version suffix.
func coreName(key string) string {
	if idx := strings.LastIndex(key, "/"); idx >= 0 {
		key = key[idx+1:]
	}
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		if prefix := key[:idx]; prefix != "" && !strings.Contains(prefix, "-") {
			key = key[idx+1:]
		}
	}
	if idx := strings.LastIndex(key, "-v"); idx > 0 {
		tail := key[idx+2:]
		if tail != "" {
			if colon := strings.Index(tail, ":"); colon > 0 {
				if allDigits(tail[:colon]) && allDigits(tail[colon+1:]) {
					key = key[:idx]
				}
			} else if allDigits(tail) {
				key = key[:idx]
			}
		}
	}
	return key
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (p *PriceBook) loadOverrides() error {
	data, err := os.ReadFile(p.overridesPath())
	if err != nil {
		return err
	}
	var parsed map[string]ModelPrice
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("overrides parse: %w", err)
	}
	p.overrides = parsed
	return nil
}

// Lookup returns the price for a model. Tries override, then exact litellm
// match, then the alias index (which maps "core" names to the cheapest
// equivalent litellm key), then the core of the queried name itself.
func (p *PriceBook) Lookup(model string) (ModelPrice, bool) {
	p.ensureFresh()
	p.mu.RLock()
	defer p.mu.RUnlock()
	if mp, ok := p.overrides[model]; ok {
		return mp, true
	}
	if mp, ok := p.prices[model]; ok {
		return mp, true
	}
	if idx := strings.Index(model, "/"); idx > 0 {
		if mp, ok := p.prices[model[idx+1:]]; ok {
			return mp, true
		}
	}
	if mp, ok := p.aliases[model]; ok {
		return mp, true
	}
	if core := coreName(model); core != model && core != "" {
		if mp, ok := p.prices[core]; ok {
			return mp, true
		}
		if mp, ok := p.aliases[core]; ok {
			return mp, true
		}
	}
	return ModelPrice{}, false
}

// Cost applies the model's prices. Missing prices → 0 and records the model
// name so operators can notice it via /unknown-models.
func (p *PriceBook) Cost(model string, input, output, cacheRead, cacheWrite, reasoning int64) float64 {
	mp, ok := p.Lookup(model)
	if !ok {
		p.mu.Lock()
		p.unknownModel[model]++
		p.mu.Unlock()
		return 0
	}
	return float64(input)*mp.InputCostPerToken +
		float64(output+reasoning)*mp.OutputCostPerToken +
		float64(cacheRead)*mp.CacheReadInputTokenCost +
		float64(cacheWrite)*mp.CacheCreationInputTokenCost
}

func (p *PriceBook) UnknownModels() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]int, len(p.unknownModel))
	for k, v := range p.unknownModel {
		out[k] = v
	}
	return out
}
