package main

import "testing"

func TestHumanTokens(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1_000, "1K"},
		{1_200, "1.2K"},
		{9_800, "9.8K"},
		{12_345, "12K"},
		{999_900, "1000K"},
		{1_000_000, "1M"},
		{1_500_000, "1.5M"},
		{15_000_000, "15M"},
		{999_000_000, "999M"},
		{1_000_000_000, "1B"},
		{2_300_000_000, "2.3B"},
		{45_600_000_000, "46B"},
		{1_000_000_000_000, "1T"},
		{3_140_000_000_000, "3.1T"},
		{-1_500_000, "-1.5M"},
	}
	for _, c := range cases {
		got := humanTokens(c.in)
		if got != c.want {
			t.Errorf("humanTokens(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
