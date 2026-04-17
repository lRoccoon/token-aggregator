#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/update_lark_token_slot.sh"

ccusage_json='{"daily":[{"date":"2026-04-16","totalTokens":0}],"totals":{"totalTokens":58979803,"totalCost":46.95273475}}'
codex_json='{"daily":[{"date":"Apr 16, 2026","totalTokens":12325720}],"totals":{"totalTokens":338375118,"costUSD":202.77933855}}'

summary="$(build_summary_text "2026-04-16" "${ccusage_json}" "${codex_json}")"
expected='已经消耗词元：今日 12M / 总计 397M，白赚 $249.73'

if [[ "${summary}" != "${expected}" ]]; then
  echo "expected: ${expected}"
  echo "actual:   ${summary}"
  exit 1
fi
