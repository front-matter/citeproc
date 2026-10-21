# Benchmarks

Numbers for spike question 2 (`plan.md` §1): is parsing CSL XML at runtime,
instead of hayagriva's precompiled CBOR, fast enough for Hugo-scale sites?

## Parsing (day 2)

Apple M1 Pro, Go 1.24.13, darwin/arm64. Median of 3 runs of
`go test -run '^$' -bench . -benchmem ./...`.

| Benchmark | Input | Time | Memory | Allocations |
|---|---|---:|---:|---:|
| Parse APA, cold (`ParseStyle`) | 85 KB | **1.9 ms** | 0.96 MB | 22,424 |
| ↳ of which XML tokenizing (`encoding/xml`) | | 1.5 ms | 0.68 MB | 17,612 |
| Load APA, cached (`LoadStyle`) | | 47 µs | 90 KB | 7 |
| Parse MLA, cold | 12 KB | 0.29 ms | 0.17 MB | 4,013 |
| Parse Vancouver, cold | 13 KB | 0.32 ms | 0.19 MB | 4,320 |
| Parse `locales-en-US.xml`, cold | 26 KB | 0.61 ms | 0.37 MB | 9,099 |
| Load `locales-en-US.xml`, cached | | 15 µs | 27 KB | 7 |
| Build locale chain (APA, en-US, files cached) | | 24 µs | 24 KB | 15 |
| Term lookup through the chain | | 28 ns | 0 | 0 |

**Result:** a cold parse of APA takes 1.9 ms, well under the 5 ms target, so
the CBOR step hayagriva uses isn't needed. A first `LoadStyle(Styles, "apa")`
plus the en-US locale costs about 2.5 ms once per process; after that,
styles and locales come from the content-keyed cache.

About 80% of the cold parse is `encoding/xml` itself. The cached path is
dominated by reading the embedded file and hashing it (SHA-256 over 85 KB);
if that ever matters, a name-keyed cache in front of it would remove it.
