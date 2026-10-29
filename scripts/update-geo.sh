#!/usr/bin/env bash
# Обновление локального индекса геокодера GeoNames (раз в месяц).
#
# Скачивает cities1000.zip, пересобирает geo/cities.idx, атомарно подменяет файл
# у astro-core и перезапускает сервис (индекс читается один раз при старте).
# Лог — ~/.local/state/astro3d-geo/, при ошибке — сообщение в Telegram.
set -uo pipefail

ROOT="/home/dmyr/swisseph_build"
TARGET="/home/dmyr/astro-core/geo/cities.idx"
STATE_DIR="${HOME}/.local/state/astro3d-geo"
mkdir -p "$STATE_DIR"
LOG="$STATE_DIR/update-$(date +%Y%m%d).log"

exec >>"$LOG" 2>&1
echo "=== $(date -Is) обновление индекса GeoNames"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fail() {
  echo "ПРОВАЛ: $1"
  token=$(grep -m1 '^TELEGRAM_BOT_TOKEN=' /home/dmyr/site-cloud/site.env 2>/dev/null | cut -d= -f2- | tr -d '"'"'"'')
  chat=$(grep -m1 '^DRILL_TG_CHAT=' /home/dmyr/pb-sidecar/backup.env 2>/dev/null | cut -d= -f2- | tr -d '"'"'"'')
  if [ -n "$token" ] && [ -n "$chat" ] && [ "${A3D_ALERT_DRY:-0}" != "1" ]; then
    curl -sf -m 15 -X POST "https://api.telegram.org/bot${token}/sendMessage" \
      -d "chat_id=${chat}" --data-urlencode "text=astro3d: обновление геокодера не удалось — $1" >/dev/null || true
  fi
  exit 1
}

curl -fsS -m 300 -o "$TMP/cities1000.zip" \
  https://download.geonames.org/export/dump/cities1000.zip || fail "не скачался дамп"

(cd "$ROOT" && go run ./cmd/geobuild -zip "$TMP/cities1000.zip" -out "$ROOT/geo/cities.idx.new") \
  || fail "не собрался индекс"

size=$(stat -c %s "$ROOT/geo/cities.idx.new" 2>/dev/null || echo 0)
[ "$size" -gt 20000000 ] || fail "индекс подозрительно мал ($size байт)"

mv "$ROOT/geo/cities.idx.new" "$ROOT/geo/cities.idx" || fail "не удалось подменить индекс"
install -m 644 "$ROOT/geo/cities.idx" "$TARGET" || fail "не удалось установить индекс для astro-core"
sudo -n /usr/bin/systemctl restart astro-core.service || fail "не удалось перезапустить astro-core"

sleep 2
code=$(curl -s -m 10 -o /dev/null -w '%{http_code}' http://127.0.0.1:8084/healthz || echo 000)
[ "$code" = "200" ] || fail "astro-core не поднялся после обновления (healthz $code)"

echo "готово: индекс $(du -h "$ROOT/geo/cities.idx" | cut -f1), astro-core отвечает"
