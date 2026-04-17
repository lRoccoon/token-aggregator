#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CRON_LOG="${ROOT_DIR}/logs/cron.log"
CRON_ENTRY="*/30 * * * * ${ROOT_DIR}/update_lark_token_slot.sh >> ${CRON_LOG} 2>&1"

mkdir -p "${ROOT_DIR}/logs"

current_crontab="$(crontab -l 2>/dev/null || true)"

if grep -Fqx "${CRON_ENTRY}" <<<"${current_crontab}"; then
  echo "cron entry already installed"
  exit 0
fi

{
  printf '%s\n' "${current_crontab}" | sed '/update_lark_token_slot\.sh/d'
  printf '%s\n' "${CRON_ENTRY}"
} | crontab -

echo "installed cron entry:"
echo "${CRON_ENTRY}"
