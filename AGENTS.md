# AGENTS.md

Go CLI that wraps the Swiss Ephemeris C library via cgo to compute astrological data (natal charts, synastry, planet periods, event calendars). Code and docs are in Russian; keep user-facing strings in Russian.

## Build & run

- There is exactly one entrypoint: `cmd/astro/main.go`. Build the whole app with:
  `go build -o astro ./cmd/astro/main.go`
- `run_calendar.sh` auto-rebuilds if any `.go` file is newer than `./astro`, then runs the `calendar` command.
- Requires cgo. Only builds with the committed static library and headers (see cgo quirk below). `CGO_ENABLED=1`.
- No lint/typecheck/CI config. Fastest verification is `go build -o /tmp/astro ./cmd/astro/main.go` plus `go vet ./...`.
- Tests: `go test ./tests/ -v`. The `tests/` package holds golden tests with the reference data embedded directly in the files:
  - `calendar_test.go` regenerates the July-2026 event calendar (UTC+0) and compares count + event list against the embedded reference `output_data/calendar_1785807837.text`, printing detailed discrepancies on mismatch.
  - `natal_test.go` regenerates the natal chart for 1971-11-19 19:43 UTC (51.77N, 55.10E, Placidus) and compares the text render against the embedded reference `output_data/natal_1785812687.text`.
  - Both need the local `ephe/` data — they skip if absent. Other packages (`pkg/astro`, `pkg/output`) have no tests.

## Calendar hot path — don't "optimize" blindly

- ~96% of `ComputeCalendar` runtime is inside `swe_calc_ut` (C library); Go-side allocations are negligible (verified with `go test -bench=CalendarYear -cpuprofile` in `tests/benchmark_test.go`).
- The golden reference pins exact minute values produced by the **current** algorithm. Some events sit within ~1–9 s of a minute boundary, so changing the golden-section iteration count (20) or the moon-loop step (1 h) flips their displayed minute and fails `TestCalendarEventsMatchReference`. Verified empirically: 17/19 iterations and 2 h step both break the test.
- To genuinely speed up the year-long calendar (~8 s), you'd need a coarse-to-fine detection rewrite that provably preserves event times — high risk, no tests beyond the golden ones.

## cgo / Swiss Ephemeris wiring

- `pkg/astro/calculator.go` declares cgo imports using a **hard-coded, relative** path:
  `#cgo CFLAGS: -I${SRCDIR}/../../../swisseph_build/include` and `LDFLAGS: -L${SRCDIR}/../../../swisseph_build/lib -lswe -lm`.
  This assumes `pkg/astro` sits under a repo named `swisseph_build` in `$GOPATH`/module dir at that depth. If the repo is moved or renamed, the build breaks — update both flags together.
- `lib/libswe.a` (Linux static lib) and headers in `include/` and `lib/` are committed. `lib/libswe_linux.zip`/`libswe_win64.zip` are upstream distributions, not used at build time.
- Ephemeris data files live in `ephe/` (gitignored; present locally). The calculator sets `swe_set_ephe_path` to the `ephe` dir next to the running binary (`GetExecutableDir()`), so a freshly built `./astro` must run from the repo root or with `ephe/` alongside it — otherwise Swiss Ephemeris silently falls back / fails to load files.
- Thread affinity: libswe.a keeps the ephe path in state that does not survive Go goroutine migration between OS threads (cgo calls trigger migration; afterwards swe silently falls back to Moshier, flipping minute-boundary events run to run). Hence `NewCalculator` does `runtime.LockOSThread()` and `Close()` unlocks — every swe call must happen between them on the same goroutine; never call swe directly outside `Calculator`.

## Output format gotcha

- `--format`/`-f` accepts `json`, `console`, `text`, `svg`, `png` (see `pkg/output/factory.go`). The README's `image` value is **not** accepted by the renderer factory.
- Image is only meaningful for `natal`/`synastry`. PNG rendering shells out to an external `resvg` binary (must be on PATH); it falls back to `./resvg` in `executeAstroJob`.

## Packages

- `cmd/astro` — cobra CLI, all subcommands (`natal`, `synastry`, `period`, `calendar`), output routing, storage cleanup wiring.
- `pkg/astro` — cgo calculator + types, aspect math (`aspects.go`), calendar event detection (`calendar*.go`), filtering (`ApplyFilter` in `types.go`).
- `pkg/interactive` — interactive CLI flows; `GetGeoData` calls **live network services** (Nominatim OSM for coords, timeapi.io for timezone). Interactive runs fail offline.
- `pkg/output` — renderers (console/text/json/svg/png).
- `pkg/storage` — background goroutine auto-deletes output files older than 7 days (`New(saveDir, 7)`, cleanup every 12h).

## Notes

- default output dir `output_data/` (gitignored), created relative to executable dir; storage cleanup runs on it.
- `do_push.sh` just pushes to `origin/main`.
- The committed `astro` binary is tracked in git; rebuilds modify it (visible in `git status`).