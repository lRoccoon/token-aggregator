#!/usr/bin/env bash
# token-collector installer.
# Flag mode:
#   curl -fsSL __SERVER_URL__/install.sh | bash -s -- --token <TOKEN>
# Interactive mode (prompts for missing token):
#   bash -c "$(curl -fsSL __SERVER_URL__/install.sh)"
set -euo pipefail

SERVER_URL="__SERVER_URL__"
TOKEN=""
DEVICE_ID=""
SOURCES="claude,codex"
INTERVAL_MIN="30"
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/token-collector"
DATA_DIR="${HOME}/.local/share/token-collector"
NO_CRON="0"
RUN_NOW="1"
ASSUME_YES="0"

usage() {
  cat <<EOF
Usage: install.sh [options]
  --token TOKEN        Bearer token (required; prompted in interactive mode)
  --server URL         Override server URL (default: ${SERVER_URL})
  --device-id ID       Device identifier (default: hostname)
  --sources CSV        Sources to collect (default: ${SOURCES})
                       built-in: claude, codex, hermes-agent
  --interval MINUTES   Cron interval (default: ${INTERVAL_MIN})
  --install-dir DIR    Install directory (default: ${INSTALL_DIR})
  --no-cron            Skip cron installation
  --no-run             Skip initial verification run
  --yes, -y            Non-interactive: use defaults, do not prompt
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --token)       TOKEN="$2"; shift 2 ;;
    --server)      SERVER_URL="$2"; shift 2 ;;
    --device-id)   DEVICE_ID="$2"; shift 2 ;;
    --sources)     SOURCES="$2"; shift 2 ;;
    --interval)    INTERVAL_MIN="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --no-cron)     NO_CRON="1"; shift ;;
    --no-run)      RUN_NOW="0"; shift ;;
    --yes|-y)      ASSUME_YES="1"; shift ;;
    -h|--help)     usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ -t 1 ]]; then
  C_GREEN=$'\e[32m'; C_YELLOW=$'\e[33m'; C_RED=$'\e[31m'; C_DIM=$'\e[2m'; C_BOLD=$'\e[1m'; C_OFF=$'\e[0m'
else
  C_GREEN=""; C_YELLOW=""; C_RED=""; C_DIM=""; C_BOLD=""; C_OFF=""
fi
info() { printf '%s==>%s %s\n' "${C_GREEN}" "${C_OFF}" "$*"; }
warn() { printf '%s[!]%s %s\n' "${C_YELLOW}" "${C_OFF}" "$*" >&2; }
err()  { printf '%s[x]%s %s\n' "${C_RED}" "${C_OFF}" "$*" >&2; }
hint() { printf '    %s%s%s\n' "${C_DIM}" "$*" "${C_OFF}"; }

INTERACTIVE=0
if [[ "${ASSUME_YES}" != "1" && -r /dev/tty ]]; then
  INTERACTIVE=1
fi

prompt() {
  local default="$1" message="$2" value
  if [[ -n "${default}" ]]; then
    printf '%s [%s]: ' "${message}" "${default}" >&2
  else
    printf '%s: ' "${message}" >&2
  fi
  IFS= read -r value </dev/tty || value=""
  if [[ -z "${value}" ]]; then
    value="${default}"
  fi
  printf '%s' "${value}"
}

detect_os() {
  if [[ "$(uname -s)" == "Darwin" ]]; then echo macos; return; fi
  if [[ -r /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    case "${ID:-}${ID_LIKE:-}" in
      *ubuntu*|*debian*)                          echo debian ;;
      *fedora*|*rhel*|*centos*|*rocky*|*alma*)    echo rhel ;;
      *alpine*)                                   echo alpine ;;
      *arch*|*manjaro*)                           echo arch ;;
      *)                                          echo linux ;;
    esac
    return
  fi
  echo unknown
}

install_cmd() {
  local os="$1" pkg="$2"
  case "${os}:${pkg}" in
    macos:node)       echo "brew install node" ;;
    macos:sqlite3)    echo "brew install sqlite" ;;
    macos:curl)       echo "brew install curl" ;;
    macos:crontab)    echo "crontab is built-in on macOS; ensure Full Disk Access for cron" ;;

    debian:node)      echo "sudo apt-get update && sudo apt-get install -y nodejs npm   # or Node 18+ via nvm / NodeSource" ;;
    debian:sqlite3)   echo "sudo apt-get install -y sqlite3" ;;
    debian:curl)      echo "sudo apt-get install -y curl" ;;
    debian:crontab)   echo "sudo apt-get install -y cron && sudo systemctl enable --now cron" ;;

    rhel:node)        echo "sudo dnf install -y nodejs npm   # or Node 18+ via nvm / NodeSource" ;;
    rhel:sqlite3)     echo "sudo dnf install -y sqlite" ;;
    rhel:curl)        echo "sudo dnf install -y curl" ;;
    rhel:crontab)     echo "sudo dnf install -y cronie && sudo systemctl enable --now crond" ;;

    alpine:node)      echo "sudo apk add --no-cache nodejs npm" ;;
    alpine:sqlite3)   echo "sudo apk add --no-cache sqlite" ;;
    alpine:curl)      echo "sudo apk add --no-cache curl" ;;
    alpine:crontab)   echo "sudo apk add --no-cache busybox-initscripts && sudo rc-update add crond" ;;

    arch:node)        echo "sudo pacman -S --needed nodejs npm" ;;
    arch:sqlite3)     echo "sudo pacman -S --needed sqlite" ;;
    arch:curl)        echo "sudo pacman -S --needed curl" ;;
    arch:crontab)     echo "sudo pacman -S --needed cronie && sudo systemctl enable --now cronie" ;;

    *)                echo "install ${pkg} via your package manager" ;;
  esac
}

OS="$(detect_os)"
info "detected OS: ${OS}"

if ! command -v curl >/dev/null; then
  err "curl is required but not installed"
  hint "$(install_cmd "${OS}" curl)"
  exit 1
fi

if [[ -z "${TOKEN}" && "${INTERACTIVE}" == "1" ]]; then
  info "interactive install (Ctrl-C to abort; use --yes to skip prompts)"
  while [[ -z "${TOKEN}" ]]; do
    TOKEN="$(prompt "" "  bearer token")"
    [[ -z "${TOKEN}" ]] && err "token cannot be empty"
  done
  DEVICE_ID="$(prompt "$(hostname -s 2>/dev/null || hostname)" "  device id")"
  SOURCES="$(prompt "${SOURCES}" "  sources (csv)")"
  INTERVAL_MIN="$(prompt "${INTERVAL_MIN}" "  cron interval (minutes, 0=disable)")"
  if [[ "${INTERVAL_MIN}" == "0" ]]; then
    NO_CRON="1"
    INTERVAL_MIN="30"
  fi
fi

if [[ -z "${TOKEN}" ]]; then
  err "--token is required (or run interactively via: bash -c \"\$(curl -fsSL ${SERVER_URL}/install.sh)\")"
  exit 1
fi
[[ -z "${DEVICE_ID}" ]] && DEVICE_ID="$(hostname -s 2>/dev/null || hostname)"

need_node=0
need_sqlite=0
IFS=',' read -r -a src_arr <<< "${SOURCES}"
for s in "${src_arr[@]}"; do
  case "${s// /}" in
    claude|codex)  need_node=1 ;;
    hermes-agent)  need_sqlite=1 ;;
  esac
done

missing=0
if [[ "${need_node}" == "1" ]] && ! command -v npx >/dev/null; then
  warn "npx / Node not found; needed by claude/codex collection"
  hint "$(install_cmd "${OS}" node)"
  missing=1
fi
if [[ "${need_sqlite}" == "1" ]] && ! command -v sqlite3 >/dev/null; then
  warn "sqlite3 not found; needed by hermes-agent collection"
  hint "$(install_cmd "${OS}" sqlite3)"
  missing=1
fi
if [[ "${NO_CRON}" != "1" ]] && ! command -v crontab >/dev/null; then
  warn "crontab not found; cron step will be skipped"
  hint "$(install_cmd "${OS}" crontab)"
  NO_CRON="1"
fi
if [[ "${missing}" == "1" ]]; then
  warn "continuing; missing deps above must be installed before the collector works end-to-end"
fi

mkdir -p "${INSTALL_DIR}" "${CONFIG_DIR}" "${DATA_DIR}"

COLLECTOR="${INSTALL_DIR}/token-collector.sh"
info "downloading collector: ${COLLECTOR}"
curl -fsSL "${SERVER_URL}/collector.sh" -o "${COLLECTOR}"
chmod +x "${COLLECTOR}"

CONFIG_FILE="${CONFIG_DIR}/config.env"
cat >"${CONFIG_FILE}" <<EOF
# token-collector config (sourced by collector.sh)
SERVER_URL="${SERVER_URL}"
TOKEN="${TOKEN}"
DEVICE_ID="${DEVICE_ID}"
SOURCES="${SOURCES}"
TIMEZONE="\${TIMEZONE:-Asia/Shanghai}"
LOG_FILE="${DATA_DIR}/collector.log"

# To add a custom source 'foo', append it to SOURCES and define:
#   adapter_foo() { your_command_emitting_standard_json; }
# Standard format: {"daily":[{"date":"YYYY-MM-DD","total_tokens":N,"cost_usd":X}]}
EOF
chmod 600 "${CONFIG_FILE}"
info "wrote config: ${CONFIG_FILE}"

if [[ "${NO_CRON}" != "1" ]]; then
  CRON_LINE="*/${INTERVAL_MIN} * * * * ${COLLECTOR} >> ${DATA_DIR}/cron.log 2>&1"
  ( crontab -l 2>/dev/null | grep -v 'token-collector.sh' || true; echo "${CRON_LINE}" ) | crontab -
  info "installed cron: ${CRON_LINE}"
else
  warn "cron not installed; run collector manually: ${COLLECTOR}"
fi

if [[ "${RUN_NOW}" == "1" ]]; then
  info "running collector once"
  if ! "${COLLECTOR}"; then
    warn "initial run returned non-zero; check ${DATA_DIR}/collector.log"
  fi
fi

printf '\n'
info "done."
printf '   %-10s %s\n' "collector" "${COLLECTOR}"
printf '   %-10s %s\n' "config" "${CONFIG_FILE}"
printf '   %-10s %s, %s\n' "logs" "${DATA_DIR}/collector.log" "${DATA_DIR}/cron.log"
printf '   %-10s %s\n' "server" "${SERVER_URL}"
printf '   %-10s %s\n' "device" "${DEVICE_ID}"
printf '   %-10s %s\n' "sources" "${SOURCES}"
