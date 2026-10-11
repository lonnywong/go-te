# Terminal allocation analysis (11 October 2026)

Screen construction now allocates one cell grid. Reset reuses it, and plain-screen index/reverse-index scrolling recycles outgoing rows. Drawing through an identity charset avoids a translated string allocation.

## Measurement

Baseline: `fc1b59e`, with the same `performance_test.go` benchmarks used for the changes. Go 1.27.1, linux/amd64, Intel N100, GOMAXPROCS=2. Commands:

```sh
make vet test
make benchmark
make profile
```

`make profile` runs the complete suite of every test-bearing package at allocation sampling rate 1. At this revision, `pkg/te` is the only such package. The other packages are compiled by `make test`. Existing upstream-derived skips remain unchanged. Normal tests and benchmarks do not capture profiles.

The baseline complete-suite profile allocated 222,823 KiB, of which 204,699 KiB (91.9%) came from blank rows. After the changes, total allocations were 108,165 KiB, with 92,128 KiB in blank rows. Most remaining row allocation comes from constructing independent screens in tests. These totals include test setup and profiler overhead; they are not live heap or embedded RAM measurements.

The short complete-suite CPU capture was noisy. A separate representative benchmark capture used:

```sh
go test ./pkg/te -run '^$' -bench . -benchtime=300ms \
  -cpuprofile="$PROJECT_TMP/runs/profiles/bench.cpu" \
  -memprofile="$PROJECT_TMP/runs/profiles/bench.mem" \
  -o "$PROJECT_TMP/build/analysis/bench.test"
```

Set `PROJECT_TMP` and cache variables to the paths defined by the Makefile before running this command. The capture identified cell filling, grapheme processing and width calculation. Cell filling now uses exponential bulk copies. Unicode segmentation and width semantics remain unchanged.

## Equivalent benchmark results

Five runs per workload; values below are medians. Allocation counts were stable. Timings varied on the shared VM, so they are indicative, not a latency guarantee.

| Workload | ns/op before → after | B/op before → after | allocs/op before → after |
|---|---:|---:|---:|
| NewScreen/40x8 | 19966 → 32722 | 57184 → 29344 | 31 → 22 |
| NewScreen/80x24 | 280579 → 178831 | 317392 → 160080 | 71 → 46 |
| NewScreen/100x30 | 621103 → 323735 | 497008 → 250480 | 85 → 54 |
| Reset | 200019 → 13423 | 159440 → 2128 | 45 → 20 |
| Scroll/up | 9025 → 552.5 | 6528 → 0 | 1 → 0 |
| Scroll/down | 8157 → 505.4 | 6528 → 0 | 1 → 0 |
| Draw/ASCII | 4229 → 2544 | 24 → 0 | 2 → 0 |
| Draw/Unicode | 6266 → 3586 | 56 → 0 | 3 → 0 |
| Display | 85289 → 50599 | 6336 → 2304 | 121 → 25 |
| Resize | 229226 → 4706 | 117369 → 577 | 36 → 12 |
| StreamScroll | 13308 → 2828 | 6630 → 80 | 4 → 2 |

A five-run follow-up of the apparent 40x8 construction slowdown measured a median of 7,088 ns/op with the same 29,344 B/op and 22 allocations. Reset measured 7,613 ns/op; scroll up/down measured 482.7/485.4 ns/op. Keep the original slower run visible: timing comparisons on this host need repeated measurements.

## Compatibility and verification

- `make vet test` passes, including race detection and 73.0% coverage of `pkg/te`.
- Existing pyte and ESCTest inputs, assertions and skips are unchanged.
- Added regressions cover row reuse, scroll margins, reverse-video blank attributes, reset, dimension clamping, resize capacity with stale-cell clearing, charset translation and malformed UTF-8.
- HistoryScreen keeps the existing outgoing-row lifetime semantics. It deliberately does not use plain-screen row recycling; the existing history tests verify this distinction.
- Dante passed `make check test-race test-browser` with a temporary local module replacement. Tests include Canvas rendering, UTF-8, keyboard/history, raw pixels, dirty updates, exit and reset. Its module files were restored afterwards.

`Screen.Buffer` is mutable screen-owned storage. Reset and scrolling can now overwrite previously obtained row slices. Consumers needing a stable snapshot should use `LinesCells()` or copy the cells. No exported method signatures or Cell fields changed.

## Remaining costs

The public pointer-bearing `Cell` layout still requires substantial grid storage. History snapshots still allocate. Grapheme segmentation and width calculation remain CPU costs. These changes reduce measured overhead without changing the terminal model or Unicode behaviour; they do not establish a minimal TinyGo RAM baseline.

Raw profiles and their analysis binaries were removed after analysis. The benchmark sources, commands, measured results and regression tests are retained. Dante's next task is a separate headless baseline including 9P and security, before any hardware tests.
