#!/usr/bin/env bash
# build-libswe.sh — собирает lib/libswe.a из вендоренных исходников Swiss Ephemeris.
#
# Раньше библиотека распаковывалась из lib/libswe_linux.zip: в публичном
# репозитории лежал готовый бинарник без Corresponding Source, что несовместимо
# с AGPL-3.0. Теперь исходники лежат в third_party/swisseph (upstream-тег
# v2.10.3final), а этот скрипт — единственный способ получить libswe.a.
#
# Набор объектов повторяет upstream-переменную SWEOBJ из Makefile:
# swedate, swehouse, swejpl, swemmoon, swemplan, sweph, swephlib, swecl, swehel.
#
#   ./scripts/build-libswe.sh
#
# Результат: lib/libswe.a (gitignored). Заголовки берутся из include/ — они
# побайтово совпадают с тем же тегом, отдельной копии в third_party нет.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
src="$root/third_party/swisseph"
inc="$root/include"
out="$root/lib"
cc="${CC:-cc}"
cflags="${CFLAGS:--O2 -Wall -fPIC}"

if [ ! -f "$src/sweph.c" ]; then
	echo "нет исходников Swiss Ephemeris в $src" >&2
	exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

objs=()
for name in swedate swehouse swejpl swemmoon swemplan sweph swephlib swecl swehel; do
	# shellcheck disable=SC2086 # CFLAGS — список флагов
	"$cc" $cflags -I"$inc" -c "$src/$name.c" -o "$tmp/$name.o"
	objs+=("$tmp/$name.o")
done

mkdir -p "$out"
rm -f "$out/libswe.a"
ar rcs "$out/libswe.a" "${objs[@]}"

echo "собрано: $out/libswe.a ($(du -h "$out/libswe.a" | cut -f1), объектов: ${#objs[@]})"
