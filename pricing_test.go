package main

import (
	"testing"
	"time"
)

func TestCoreName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20241022"},
		{"anthropic.claude-3-5-sonnet-20241022-v2:0", "claude-3-5-sonnet-20241022"},
		{"us.anthropic.claude-3-5-sonnet-20241022-v2:0", "claude-3-5-sonnet-20241022"},
		{"bedrock/us.anthropic.claude-3-5-sonnet-20241022-v2:0", "claude-3-5-sonnet-20241022"},
		{"vercel_ai_gateway/anthropic/claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20241022"},
		{"openrouter/anthropic/claude-3.5-sonnet", "claude-3.5-sonnet"},
		{"gpt-5.4", "gpt-5.4"},
		{"gemini-1.5-pro", "gemini-1.5-pro"},
		{"claude-3-5-haiku-latest", "claude-3-5-haiku-latest"},
	}
	for _, c := range cases {
		if got := coreName(c.in); got != c.want {
			t.Errorf("coreName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLookupAliases(t *testing.T) {
	pb := NewPriceBook(t.TempDir())
	pb.prices = map[string]ModelPrice{
		"anthropic.claude-3-5-sonnet-20241022-v2:0": {
			InputCostPerToken:           0.000003,
			OutputCostPerToken:          0.000015,
			CacheReadInputTokenCost:     0.0000003,
			CacheCreationInputTokenCost: 0.00000375,
		},
		"gpt-5.4": {
			InputCostPerToken:  0.00000125,
			OutputCostPerToken: 0.000010,
		},
	}
	pb.aliases = buildAliases(pb.prices)
	pb.loadedAt = nowForTest()

	if mp, ok := pb.Lookup("claude-3-5-sonnet-20241022"); !ok || mp.InputCostPerToken == 0 {
		t.Fatalf("expected alias hit for claude-3-5-sonnet-20241022, got ok=%v mp=%+v", ok, mp)
	}
	if mp, ok := pb.Lookup("gpt-5.4"); !ok || mp.InputCostPerToken == 0 {
		t.Fatalf("expected exact hit for gpt-5.4, got ok=%v mp=%+v", ok, mp)
	}
	if _, ok := pb.Lookup("nonexistent-model"); ok {
		t.Fatalf("expected miss for nonexistent-model")
	}
}

func nowForTest() time.Time { return time.Now().Add(1 * time.Hour) }
