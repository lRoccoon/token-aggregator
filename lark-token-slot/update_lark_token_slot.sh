#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="${ROOT_DIR}/logs"
LOG_FILE="${LOG_DIR}/update.log"
ONLINE_REFRESH_MARKER="${LOG_DIR}/.last_online_refresh"

# Source optional config file — used by cron (which has no inherited env).
# Layout: LARK_SLOT_ID / LARK_SLOT_CREDENTIAL / [LARK_SLOT_API_URL] / [LARK_SLOT_INFO_URL] / [CCUSAGE_BIN]
CONFIG_FILE="${LARK_TOKEN_SLOT_CONFIG:-${HOME}/.config/lark-token-slot/config.env}"
if [[ -r "${CONFIG_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${CONFIG_FILE}"
fi

readonly SLOT_ID="${LARK_SLOT_ID:?LARK_SLOT_ID env var required (set it in ${CONFIG_FILE})}"
readonly CREDENTIAL="${LARK_SLOT_CREDENTIAL:?LARK_SLOT_CREDENTIAL env var required (set it in ${CONFIG_FILE})}"
readonly SLOT_API_URL="${LARK_SLOT_API_URL:-https://l.garyyang.work/api/slot/update}"
readonly SLOT_INFO_URL="${LARK_SLOT_INFO_URL:-https://l.garyyang.work/api/slot/info}"
readonly CCUSAGE_BIN="${CCUSAGE_BIN:-npx}"
readonly FETCH_TIMEOUT_SECONDS="${FETCH_TIMEOUT_SECONDS:-180}"

require_bin() {
  local bin="$1"
  command -v "${bin}" >/dev/null 2>&1 || {
    echo "missing dependency: ${bin}" >&2
    return 1
  }
}

format_millions() {
  local tokens="$1"
  echo "$((tokens / 1000000))M"
}

build_summary_text() {
  local today_iso="$1"
  local ccusage_json="$2"
  local codex_json="$3"
  local today_codex
  local ccusage_today
  local ccusage_total
  local ccusage_cost
  local codex_today
  local codex_total
  local codex_cost
  local total_today
  local total_tokens
  local total_cost

  today_codex="$(LC_ALL=C TZ=Asia/Shanghai date -d "${today_iso}" '+%b %-d, %Y')"

  ccusage_today="$(jq -r --arg today "${today_iso}" '[.daily[] | select(.date == $today) | .totalTokens] | add // 0' <<<"${ccusage_json}")"
  ccusage_total="$(jq -r '.totals.totalTokens // 0' <<<"${ccusage_json}")"
  ccusage_cost="$(jq -r '.totals.totalCost // 0' <<<"${ccusage_json}")"

  codex_today="$(jq -r --arg today "${today_codex}" '[.daily[] | select(.date == $today) | .totalTokens] | add // 0' <<<"${codex_json}")"
  codex_total="$(jq -r '.totals.totalTokens // 0' <<<"${codex_json}")"
  codex_cost="$(jq -r '.totals.costUSD // 0' <<<"${codex_json}")"

  total_today="$((ccusage_today + codex_today))"
  total_tokens="$((ccusage_total + codex_total))"
  total_cost="$(jq -nr --arg a "${ccusage_cost}" --arg b "${codex_cost}" '($a | tonumber) + ($b | tonumber)')"

  printf '已经消耗词元：今日 %s / 总计 %s，白赚 $%.2f' \
    "$(format_millions "${total_today}")" \
    "$(format_millions "${total_tokens}")" \
    "${total_cost}"
}

fetch_ccusage_json() {
  local offline="$1"
  local args=(-y ccusage daily -j -z Asia/Shanghai)
  [[ "${offline}" == "true" ]] && args+=(-O)
  timeout "${FETCH_TIMEOUT_SECONDS}" "${CCUSAGE_BIN}" "${args[@]}"
}

fetch_codex_json() {
  local offline="$1"
  local args=(-y @ccusage/codex@latest daily -j -z Asia/Shanghai)
  [[ "${offline}" == "true" ]] && args+=(-O)
  timeout "${FETCH_TIMEOUT_SECONDS}" "${CCUSAGE_BIN}" "${args[@]}"
}

should_refresh_online() {
  local today_iso="$1"
  [[ ! -f "${ONLINE_REFRESH_MARKER}" ]] && return 0
  local last
  last="$(cat "${ONLINE_REFRESH_MARKER}" 2>/dev/null || true)"
  [[ "${last}" != "${today_iso}" ]]
}

mark_online_refreshed() {
  local today_iso="$1"
  mkdir -p "${LOG_DIR}"
  printf '%s\n' "${today_iso}" >"${ONLINE_REFRESH_MARKER}"
}

write_slot() {
  local summary="$1"
  curl -sS "${SLOT_API_URL}" \
    -H "Authorization: Bearer ${CREDENTIAL}" \
    -H "Content-Type: application/json" \
    -d "$(jq -cn --arg slotId "${SLOT_ID}" --arg value "${summary}" '{slotId: $slotId, value: $value}')"
}

query_slot_info() {
  curl -sS "${SLOT_INFO_URL}" \
    -H "Authorization: Bearer ${CREDENTIAL}"
}

log_line() {
  local message="$1"
  mkdir -p "${LOG_DIR}"
  printf '[%s] %s\n' "$(TZ=Asia/Shanghai date '+%F %T %Z')" "${message}" >>"${LOG_FILE}"
}

main() {
  local mode="${1:-}"
  local today_iso
  local ccusage_json
  local codex_json
  local summary
  local response
  local slot_info
  local fetch_mode
  local offline_flag
  local update_result

  require_bin jq
  require_bin curl
  require_bin timeout
  today_iso="$(TZ=Asia/Shanghai date +%F)"

  if should_refresh_online "${today_iso}"; then
    fetch_mode="online"
    offline_flag="false"
  else
    fetch_mode="offline"
    offline_flag="true"
  fi

  ccusage_json="$(fetch_ccusage_json "${offline_flag}")"
  codex_json="$(fetch_codex_json "${offline_flag}")"
  [[ "${fetch_mode}" == "online" ]] && mark_online_refreshed "${today_iso}"
  summary="$(build_summary_text "${today_iso}" "${ccusage_json}" "${codex_json}")"

  if [[ "${mode}" == "--dry-run" ]]; then
    printf '%s\n' "${summary}"
    return 0
  fi

  response="$(write_slot "${summary}")"
  printf '%s\n' "${response}"

  if [[ "$(jq -r '.success // false' <<<"${response}" 2>/dev/null)" == "true" ]]; then
    update_result="success"
  else
    update_result="failed"
  fi

  slot_info="$(query_slot_info 2>/dev/null || echo '{}')"
  log_line "date=${today_iso} mode=${fetch_mode} result=${update_result} summary=${summary} response=${response} slot_info=${slot_info}"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "${@:-}"
fi
