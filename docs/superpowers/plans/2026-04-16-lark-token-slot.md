# Lark Token Slot Automation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local automation script that aggregates `ccusage` and `@ccusage/codex` usage, writes a formatted summary into a Lark custom slot, and refreshes it every 30 minutes via cron.

**Architecture:** Use one Bash script as the single entrypoint, with small shell functions for dependency checks, JSON aggregation, text formatting, and slot updates. Keep credentials and target slot IDs in the script for now because this is a single-user personal automation living under `~/Code`.

**Tech Stack:** Bash, `jq`, `curl`, `npx`, `ccusage`, `@ccusage/codex`, user `crontab`

---

### Task 1: Create the failing shell test

**Files:**
- Create: `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/tests/test_update_lark_token_slot.sh`
- Test: `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/tests/test_update_lark_token_slot.sh`

- [ ] **Step 1: Write the failing test**

```bash
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `bash /data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/tests/test_update_lark_token_slot.sh`
Expected: FAIL because `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/update_lark_token_slot.sh` does not exist yet.

### Task 2: Implement the updater script

**Files:**
- Create: `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/update_lark_token_slot.sh`
- Modify: `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/tests/test_update_lark_token_slot.sh`

- [ ] **Step 1: Write minimal implementation**

```bash
#!/usr/bin/env bash
set -euo pipefail

TODAY_ISO="$(TZ=Asia/Shanghai date +%F)"
TODAY_CODEX="$(TZ=Asia/Shanghai date '+%b %-d, %Y')"

build_summary_text() {
  local today_iso="$1"
  local ccusage_json="$2"
  local codex_json="$3"

  # jq logic here to sum today tokens, total tokens, and total cost
  # final format:
  # 已经消耗词元：今日 12M / 总计 397M，白赚 $249.73
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `bash /data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/tests/test_update_lark_token_slot.sh`
Expected: PASS with no output.

### Task 3: Wire real data fetching and slot update

**Files:**
- Modify: `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/update_lark_token_slot.sh`

- [ ] **Step 1: Add real command execution**

Commands that the script must run:

```bash
/home/linuxbrew/.linuxbrew/bin/npx -y ccusage daily -j -O -z Asia/Shanghai
/home/linuxbrew/.linuxbrew/bin/npx -y @ccusage/codex@latest daily -j -z Asia/Shanghai
curl -sS 'https://l.garyyang.work/api/slot/update' \
  -H 'Authorization: Bearer <credential>' \
  -H 'Content-Type: application/json' \
  -d '{"slotId":"<slot_id>","value":"<summary>"}'
```

- [ ] **Step 2: Verify with a dry local summary**

Run: `bash /data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/update_lark_token_slot.sh --dry-run`
Expected: one line summary printed to stdout, no API write.

- [ ] **Step 3: Verify end-to-end write**

Run: `bash /data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/update_lark_token_slot.sh`
Expected: API returns success JSON and the log records the written summary.

### Task 4: Install cron

**Files:**
- Create: `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/install_cron.sh`

- [ ] **Step 1: Write idempotent cron installer**

Cron entry:

```cron
*/30 * * * * /data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/update_lark_token_slot.sh >> /data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/logs/cron.log 2>&1
```

- [ ] **Step 2: Install and verify**

Run: `bash /data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/install_cron.sh`
Expected: `crontab -l` contains exactly one matching entry.

### Task 5: Create the preview template reference

**Files:**
- Create: `/data00/home/wanghaoyu.ff/Code/personal-automation/lark-token-slot/README.md`

- [ ] **Step 1: Document the template**

Template content:

```handlebars
已经消耗词元：{{slot id="<YOUR_SLOT_ID>"}}
```

- [ ] **Step 2: Record both preview URLs**

Direct link and editor link must both be included in the README.
