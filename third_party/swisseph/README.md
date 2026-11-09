# Swiss Ephemeris 2.10.03 (vendored)

Исходники C-библиотеки Swiss Ephemeris, из которых собирается `lib/libswe.a`.
Вендорены, чтобы репозиторий содержал Corresponding Source по AGPL-3.0, а не
только готовую статическую библиотеку.

- Upstream: <https://github.com/aloistr/swisseph>
- Тег: `v2.10.3final`
- Commit: `af9823fe7b06ffefe3d3968fdc5680be8b5eec5f`
- `SE_VERSION` в `include/sweph.h`: `2.10.03`
- Лицензия: AGPL-3.0 (`agpl-3.0.txt`), условия двойного лицензирования — `LICENSE`

## Что именно вендорено

Только объекты, входящие в `libswe.a` по upstream-`Makefile` (`SWEOBJ`):

```
swedate.c  swehouse.c  swejpl.c  swemmoon.c  swemplan.c
sweph.c    swephlib.c  swecl.c   swehel.c
```

Заголовки лежат в `include/` в корне репозитория и **побайтово совпадают** с тегом
`v2.10.3final`; второй копии здесь намеренно нет, чтобы версии не разъезжались.

Не вендорим `swetest.c`, `swemini.c`, `swevents.c`, `sweasp.c`, `sweclips.c`,
`sweephe4.c`, `setest/`, `doc/`, `swewin/`: они не входят в библиотеку, которую
мы распространяем. Если понадобится — берутся из того же тега.

Файлы эфемерид (`sepl_*.se1`, `semo_*.se1`, `seas_*.se1`) — данные Astrodienst,
в репозитории не хранятся, лежат в `/home/dmyr/app/data/ephe` (см. `NOTICE`).

## Сборка

```bash
./scripts/build-libswe.sh     # → lib/libswe.a (в git не хранится)
```

Скрипт компилирует ровно эти девять файлов с `-O2 -Wall -fPIC` и `-Iinclude`,
как upstream-`Makefile`, и складывает объекты в архив.
