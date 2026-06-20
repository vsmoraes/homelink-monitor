#!/bin/sh
set -eu

PKG_NAME="homelink-monitor"
PKG_VAR="${SYNOPKG_PKGVAR:-/var/packages/$PKG_NAME/var}"
OUTBOX_DIR="$PKG_VAR/events/outbox"
PROCESSED_DIR="$PKG_VAR/events/processed"
FAILED_DIR="$PKG_VAR/events/failed"
LOG_DIR="$PKG_VAR/logs"
LOG_FILE="$LOG_DIR/notification-helper.log"
POLL_SECONDS="${NOTIFICATION_POLL_SECONDS:-5}"

mkdir -p "$OUTBOX_DIR" "$PROCESSED_DIR" "$FAILED_DIR" "$LOG_DIR"

log() {
  printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*" >> "$LOG_FILE"
}

json_value() {
  key="$1"
  file="$2"
  if command -v jq >/dev/null 2>&1; then
    jq -r --arg key "$key" '.[$key] // empty' "$file" 2>/dev/null
    return 0
  fi
  sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "$file" | head -n 1
}

valid_event() {
  file="$1"
  if command -v jq >/dev/null 2>&1; then
    jq -e '.id and .severity and .metric and .title and .message and .timestamp' "$file" >/dev/null 2>&1
    return $?
  fi
  grep -q '"id"' "$file" &&
    grep -q '"severity"' "$file" &&
    grep -q '"metric"' "$file" &&
    grep -q '"title"' "$file" &&
    grep -q '"message"' "$file" &&
    grep -q '"timestamp"' "$file"
}

send_dsm_notification() {
  title="$1"
  message="$2"

  if command -v synodsmnotify >/dev/null 2>&1; then
    synodsmnotify @administrators "$title" "$message" >/dev/null 2>&1
    return $?
  fi
  if [ -x /usr/syno/bin/synodsmnotify ]; then
    /usr/syno/bin/synodsmnotify @administrators "$title" "$message" >/dev/null 2>&1
    return $?
  fi
  if command -v synonotify >/dev/null 2>&1; then
    synonotify "$title" "$message" >/dev/null 2>&1
    return $?
  fi

  log "DSM notification command not available; title=$title message=$message"
  return 0
}

process_event() {
  file="$1"
  base="$(basename "$file")"

  if ! valid_event "$file"; then
    log "invalid event file: $base"
    mv "$file" "$FAILED_DIR/$base" 2>/dev/null || rm -f "$file"
    return 0
  fi

  severity="$(json_value severity "$file")"
  metric="$(json_value metric "$file")"
  title="$(json_value title "$file")"
  message="$(json_value message "$file")"

  if send_dsm_notification "[$severity][$metric] $title" "$message"; then
    log "processed event: $base"
    mv "$file" "$PROCESSED_DIR/$base" 2>/dev/null || rm -f "$file"
    return 0
  fi

  log "failed to send event: $base"
  mv "$file" "$FAILED_DIR/$base" 2>/dev/null || rm -f "$file"
}

log "notification helper started"

while :; do
  for file in "$OUTBOX_DIR"/*.json; do
    [ -e "$file" ] || continue
    process_event "$file"
  done
  sleep "$POLL_SECONDS"
done
