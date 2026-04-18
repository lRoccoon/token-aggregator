#!/usr/bin/env bash
# token-collector: reads per-source usage and POSTs it to the central aggregator.
# Core deps: bash, curl. Optional: jq (richer log lines), npx (claude/codex),
# sqlite3 (hermes-agent). cron runs with a minimal PATH, so we prepend common
# install prefixes up front and let the user override binary paths via config.
set -u
set -o pipefail

CONFIG_FILE="${TOKEN_COLLECTOR_CONFIG:-${HOME}/.config/token-collector/config.env}"
if [[ ! -r "${CONFIG_FILE}" ]]; then
  echo "config not found: ${CONFIG_FILE}" >&2
  exit 1
fi
# shellcheck disable=SC1090
source "${CONFIG_FILE}"

: "${SERVER_URL:?SERVER_URL missing in config}"
: "${TOKEN:?TOKEN missing in config}"
: "${DEVICE_ID:?DEVICE_ID missing in config}"
: "${SOURCES:=claude,codex}"
: "${TIMEZONE:=Asia/Shanghai}"
: "${LOG_FILE:=${HOME}/.local/share/token-collector/collector.log}"

for d in "${HOME}/.local/bin" /opt/homebrew/bin /home/linuxbrew/.linuxbrew/bin /usr/local/bin; do
  [[ -d "${d}" ]] || continue
  case ":${PATH}:" in *":${d}:"*) ;; *) PATH="${d}:${PATH}" ;; esac
done
export PATH

NPX_BIN="${NPX_BIN:-$(command -v npx 2>/dev/null || true)}"
SQLITE3_BIN="${SQLITE3_BIN:-$(command -v sqlite3 2>/dev/null || true)}"
JQ_BIN="${JQ_BIN:-$(command -v jq 2>/dev/null || true)}"

mkdir -p "$(dirname "${LOG_FILE}")"

now_ts() { TZ="${TIMEZONE}" date '+%F %T %Z'; }
log() { printf '[%s] %s\n' "$(now_ts)" "$*" >>"${LOG_FILE}"; }

format_for() {
  case "$1" in
    claude)       echo ccusage ;;
    codex)        echo codex ;;
    hermes-agent) echo hermes ;;
    *)            echo standard ;;
  esac
}

fmt_tokens() {
  local n="${1:-0}"
  if ! [[ "${n}" =~ ^[0-9]+$ ]]; then
    printf '?'
    return
  fi
  if (( n >= 1000000 )); then
    awk -v n="${n}" 'BEGIN{ printf "%.1fM", n/1000000 }'
  elif (( n >= 1000 )); then
    awk -v n="${n}" 'BEGIN{ printf "%.1fK", n/1000 }'
  else
    printf '%d' "${n}"
  fi
}

# Echoes "<today_tokens> <total_tokens>". Uses jq when available; otherwise "? ?".
payload_summary() {
  local format="$1" payload="$2" today_iso today_codex
  if [[ -z "${JQ_BIN}" ]]; then
    echo "? ?"
    return
  fi
  today_iso="$(TZ="${TIMEZONE}" date +%F)"
  today_codex="$(LC_ALL=C TZ="${TIMEZONE}" date +'%b %-d, %Y' 2>/dev/null || echo "")"
  case "${format}" in
    ccusage)
      "${JQ_BIN}" -r --arg d "${today_iso}" '
        ( [.daily[]? | select(.date == $d) | .totalTokens] | add // 0 ) as $t |
        ( .totals.totalTokens // 0 ) as $tot |
        "\($t) \($tot)"
      ' <<<"${payload}" 2>/dev/null || echo "? ?"
      ;;
    codex)
      "${JQ_BIN}" -r --arg d "${today_codex}" '
        ( [.daily[]? | select(.date == $d) | .totalTokens] | add // 0 ) as $t |
        ( .totals.totalTokens // 0 ) as $tot |
        "\($t) \($tot)"
      ' <<<"${payload}" 2>/dev/null || echo "? ?"
      ;;
    hermes)
      "${JQ_BIN}" -r --arg d "${today_iso}" '
        def sum_tokens: (.input_tokens // 0) + (.output_tokens // 0) + (.cache_read_tokens // 0) + (.cache_write_tokens // 0) + (.reasoning_tokens // 0);
        ( [.daily_by_model[]? | select(.date == $d) | sum_tokens] | add // 0 ) as $t |
        ( [.daily_by_model[]? | sum_tokens] | add // 0 ) as $tot |
        "\($t) \($tot)"
      ' <<<"${payload}" 2>/dev/null || echo "? ?"
      ;;
    standard)
      "${JQ_BIN}" -r --arg d "${today_iso}" '
        ( [.daily[]? | select(.date == $d) | .total_tokens] | add // 0 ) as $t |
        ( [.daily[]? | .total_tokens] | add // 0 ) as $tot |
        "\($t) \($tot)"
      ' <<<"${payload}" 2>/dev/null || echo "? ?"
      ;;
    *) echo "? ?" ;;
  esac
}

POST_BODY=""
post_payload() {
  local source="$1" format="$2" payload="$3"
  local tmp code
  tmp="$(mktemp)"
  code="$(curl -sS -o "${tmp}" -w '%{http_code}' \
    -X POST "${SERVER_URL}/ingest" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "Content-Type: application/json" \
    -H "X-Device-Id: ${DEVICE_ID}" \
    -H "X-Source: ${source}" \
    -H "X-Format: ${format}" \
    -H "X-Timezone: ${TIMEZONE}" \
    --data-binary "${payload}")" || code="000"
  POST_BODY="$(cat "${tmp}" 2>/dev/null || true)"
  rm -f "${tmp}"
  printf '%s' "${code}"
}

ADAPTER_SKIP_REASON=""

adapter_claude() {
  [[ -z "${NPX_BIN}" ]] && { ADAPTER_SKIP_REASON="npx not found (set NPX_BIN in ${CONFIG_FILE})"; return 1; }
  "${NPX_BIN}" -y ccusage daily -j -O -z "${TIMEZONE}" 2>/dev/null
}

adapter_codex() {
  [[ -z "${NPX_BIN}" ]] && { ADAPTER_SKIP_REASON="npx not found (set NPX_BIN in ${CONFIG_FILE})"; return 1; }
  "${NPX_BIN}" -y @ccusage/codex@latest daily -j -O -z "${TIMEZONE}" 2>/dev/null
}

# hermes-agent: aggregate ~/.hermes/state.db sessions by (day, model) and let the
# server apply LiteLLM pricing. Hermes itself stores tokens per session but no
# reliable cost column, so we ship the raw token breakdown.
# Overrides via config.env: HERMES_STATE_DB (path), HERMES_DAYS (lookback window).
adapter_hermes_agent() {
  local db="${HERMES_STATE_DB:-${HOME}/.hermes/state.db}"
  local days="${HERMES_DAYS:-30}"
  if [[ ! -r "${db}" ]]; then
    ADAPTER_SKIP_REASON="state db not readable: ${db}"
    return 1
  fi
  if [[ -z "${SQLITE3_BIN}" ]]; then
    ADAPTER_SKIP_REASON="sqlite3 not found (set SQLITE3_BIN in ${CONFIG_FILE})"
    return 1
  fi
  local rows
  if ! rows="$(TZ="${TIMEZONE}" "${SQLITE3_BIN}" -separator $'\t' "${db}" \
    "SELECT date(started_at,'unixepoch','localtime'),
            COALESCE(model,'unknown'),
            SUM(COALESCE(input_tokens,0)),
            SUM(COALESCE(output_tokens,0)),
            SUM(COALESCE(cache_read_tokens,0)),
            SUM(COALESCE(cache_write_tokens,0)),
            SUM(COALESCE(reasoning_tokens,0))
     FROM sessions
     WHERE started_at >= strftime('%s','now','-${days} days')
     GROUP BY 1, 2 ORDER BY 1 DESC, 2;" 2>/dev/null)"; then
    ADAPTER_SKIP_REASON="sqlite3 query failed on ${db}"
    return 1
  fi
  printf '{"daily_by_model":['
  local first=1 day model input output cread cwrite reason
  while IFS=$'\t' read -r day model input output cread cwrite reason; do
    [[ -z "${day}" ]] && continue
    if [[ ${first} -eq 1 ]]; then first=0; else printf ','; fi
    model="${model//\\/}"
    model="${model//\"/}"
    printf '{"date":"%s","model":"%s","input_tokens":%s,"output_tokens":%s,"cache_read_tokens":%s,"cache_write_tokens":%s,"reasoning_tokens":%s}' \
      "${day}" "${model}" "${input:-0}" "${output:-0}" "${cread:-0}" "${cwrite:-0}" "${reason:-0}"
  done <<< "${rows}"
  printf ']}'
}

run_source() {
  local source="$1" fn format payload code today total today_fmt total_fmt body_snippet
  case "${source}" in
    claude)       fn=adapter_claude ;;
    codex)        fn=adapter_codex ;;
    hermes-agent) fn=adapter_hermes_agent ;;
    *)
      fn="adapter_${source//-/_}"
      if ! declare -F "${fn}" >/dev/null 2>&1; then
        log "  ${source}: SKIPPED (no adapter function ${fn})"
        return 1
      fi
      ;;
  esac
  format="$(format_for "${source}")"
  ADAPTER_SKIP_REASON=""
  if ! payload="$("${fn}")"; then
    log "  ${source}: SKIPPED (${ADAPTER_SKIP_REASON:-adapter failed})"
    return 1
  fi
  if [[ -z "${payload}" ]]; then
    log "  ${source}: SKIPPED (empty payload)"
    return 1
  fi
  read -r today total <<< "$(payload_summary "${format}" "${payload}")"
  today_fmt="$(fmt_tokens "${today}")"
  total_fmt="$(fmt_tokens "${total}")"
  code="$(post_payload "${source}" "${format}" "${payload}")"
  if [[ "${code}" == "200" ]]; then
    log "  ${source}: today=${today_fmt} total=${total_fmt} http=${code} OK"
    return 0
  fi
  body_snippet="${POST_BODY//$'\n'/ }"
  body_snippet="${body_snippet:0:200}"
  log "  ${source}: today=${today_fmt} total=${total_fmt} http=${code} FAIL response=${body_snippet}"
  return 1
}

start_ts=$(date +%s)
log "run start device=${DEVICE_ID} sources=${SOURCES} npx=${NPX_BIN:-missing} sqlite3=${SQLITE3_BIN:-missing} jq=${JQ_BIN:-missing}"

IFS=',' read -r -a src_arr <<< "${SOURCES}"
ok=0
fail=0
for s in "${src_arr[@]}"; do
  s="${s// /}"
  [[ -z "${s}" ]] && continue
  if run_source "${s}"; then
    ok=$((ok + 1))
  else
    fail=$((fail + 1))
  fi
done
end_ts=$(date +%s)
log "run end: ok=${ok} fail=${fail} elapsed=$((end_ts - start_ts))s"
exit $(( fail > 0 ? 1 : 0 ))
