# go-te

Read docs/SPEC.md for emulator scope. Preserve existing pyte/ESCTest inputs and assertions; follow docs/PYTE_PORTING_CONVENTION.md and docs/ESCTEST_PORTING_CONVENTION.md for ports. Screen/Stream are not thread-safe. No process, PTY or keyboard driver belongs in the library.

Use `make vet`, `make test`, `make coverage` and `make benchmark`. Profiling is for explicit pre-release analysis: `make profile` profiles every test-bearing package's complete suite. Do not attach profiling to ordinary tests. Optimise measured workloads; re-run correctness tests and equivalent benchmarks. Retain concise measured findings/toolchain identities, then delete raw profiles and analysis binaries after analysis.

## Disposable files

The Makefile snapshots inherited TMPDIR before redirecting it. Base priority: explicit WORKSPACE_TMP_BASE; in CI, RUNNER_TEMP then inherited TMPDIR then /tmp; locally, writable /workspace/tmp then inherited TMPDIR then /tmp. Unset/empty/0/false CI values mean local mode. Always append go-te.

Layout: cache/go-build (GOCACHE), cache/go-mod (GOMODCACHE), build/go-tmp (GOTMPDIR), build/analysis, tests (coverage.out), logs and runs (TMPDIR), with raw profiles under runs/profiles. GOTOOLCHAIN=local keeps toolchain selection explicit. Linked worktrees set WORKSPACE_TMP_BASE=<base>/go-te/worktrees/<name>, producing <base>/go-te/worktrees/<name>/go-te. Source, credentials and curated documentation remain outside disposable storage. Do not clean active caches or unrelated data. `make clean` removes this project's owned build/test outputs only.

Read relevant source before editing. Tests belong beside the implementation. Do not change shared module caches. Use a temporary consumer module replacement only during local validation and remove it before publishing the consumer's changes.
