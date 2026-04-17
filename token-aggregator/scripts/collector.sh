#!/usr/bin/env bash
# token-collector: reads per-source usage and POSTs it to the central aggregator.
# Core deps: bash, curl. Per-source deps loaded only when that source is enabled
# (claude/codex need npx/node; hermes-agent uses HERMES_CMD provided by user).
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

mkdir -p "$(dirname "${LOG_FILE}")"

log() {
  printf '[%s] %s\n' "$(TZ="${TIMEZONE}" date '+%F %T %Z')" "$*" >>"${LOG_FILE}"
}

format_for() {
  case "$1" in
    claude)       echo ccusage ;;
    codex)        echo codex ;;
    hermes-agent) echo hermes ;;
    *)            echo standard ;;
  esac
}

post_payload() {
  local source="$1" format="$2" payload="$3"
  local tmp code body
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
  body="$(cat "${tmp}" 2>/dev/null || true)"
  rm -f "${tmp}"
  log "source=${source} format=${format} http=${code} response=${body}"
  [[ "${code}" == "200" ]]
}

adapter_claude() {
  command -v npx >/dev/null || { log "skip claude: npx not found"; return 1; }
  npx -y ccusage daily -j -O -z "${TIMEZONE}" 2>/dev/null
}

adapter_codex() {
  command -v npx >/dev/null || { log "skip codex: npx not found"; return 1; }
  npx -y @ccusage/codex@latest daily -j -O -z "${TIMEZONE}" 2>/dev/null
}

# hermes-agent: aggregate ~/.hermes/state.db sessions by (day, model) and let the
# server apply LiteLLM pricing. Hermes itself stores tokens per session but no
# reliable cost column, so we ship the raw token breakdown.
# Overrides via config.env: HERMES_STATE_DB (path), HERMES_DAYS (lookback window).
adapter_hermes_agent() {
  local db="${HERMES_STATE_DB:-${HOME}/.hermes/state.db}"
  local days="${HERMES_DAYS:-30}"
  if [[ ! -r "${db}" ]]; then
    log "skip hermes-agent: ${db} not readable"
    return 1
  fi
  if ! command -v sqlite3 >/dev/null; then
    log "skip hermes-agent: sqlite3 not found"
    return 1
  fi
  local rows
  if ! rows="$(TZ="${TIMEZONE}" sqlite3 -separator $'\t' "${db}" \
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
    log "hermes-agent: sqlite3 query failed"
    return 1
  fi
  printf '{"daily_by_model":['
  local first=1 day model input output cread cwrite reason
  while IFS=$'\t' read -r day model input output cread cwrite reason; do
    [[ -z "${day}" ]] && continue
    if [[ ${first} -eq 1 ]]; then first=0; else printf ','; fi
    # model comes from SQLite raw text — strip " and \ to keep JSON valid.
    model="${model//\\/}"
    model="${model//\"/}"
    printf '{"date":"%s","model":"%s","input_tokens":%s,"output_tokens":%s,"cache_read_tokens":%s,"cache_write_tokens":%s,"reasoning_tokens":%s}' \
      "${day}" "${model}" "${input:-0}" "${output:-0}" "${cread:-0}" "${cwrite:-0}" "${reason:-0}"
  done <<< "${rows}"
  printf ']}'
}

run_source() {
  local source="$1" fn payload
  case "${source}" in
    claude)       fn=adapter_claude ;;
    codex)        fn=adapter_codex ;;
    hermes-agent) fn=adapter_hermes_agent ;;
    *)
      fn="adapter_${source//-/_}"
      if ! declare -F "${fn}" >/dev/null 2>&1; then
        log "skip ${source}: no adapter function ${fn}"
        return 1
      fi
      ;;
  esac
  if ! payload="$("${fn}")"; then
    log "adapter failed: source=${source}"
    return 1
  fi
  if [[ -z "${payload}" ]]; then
    log "adapter empty payload: source=${source}"
    return 1
  fi
  post_payload "${source}" "$(format_for "${source}")" "${payload}"
}

IFS=',' read -r -a src_arr <<< "${SOURCES}"
rc=0
for s in "${src_arr[@]}"; do
  s="${s// /}"
  [[ -z "${s}" ]] && continue
  if ! run_source "${s}"; then
    rc=1
  fi
done
exit "${rc}"
