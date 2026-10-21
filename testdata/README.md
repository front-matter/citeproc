# testdata

Test fixtures and test lists for citeproc. See `docs/test-inventory.md` for
how the lists were produced.

## Pinned sources

| Source | Commit | Licence | Use here |
|---|---|---|---|
| [hayagriva](https://github.com/typst/hayagriva) 0.10.1 | `e51299ecd1a9f62b2f069af2f554ae4e84d50527` | MIT OR Apache-2.0 | Pass list, local fixtures and unit tests are ported from it |
| [CSL test suite](https://github.com/citation-style-language/test-suite) | `6eefc5b07c6969ab8999e48542acbcc131cba864` (2026-04-08) | **None stated** (no licence file; GitHub reports no licence) | Fetched at this commit for test runs, **not vendored** |
| [CSL schema](https://github.com/citation-style-language/schema) v1.0.2 | `506040022f6c37846343edd36658b23e85b5b8ff` | MIT | Term, variable and type tables in `internal/csl/taxonomy.go` |
| [CSL locales](https://github.com/citation-style-language/locales) | `8bef426f89f38c0789223c50e53ec2197bc50bb3` (2026-10-20) | CC BY-SA 3.0 | Primary-dialect table in `internal/locale/code.go`; parser corpus test |
| [CSL styles](https://github.com/citation-style-language/styles) | `45ded9ce7b9ca726e7675b4a8976748eec9e3707` (2026-10-21) | CC BY-SA 3.0 | Parser corpus test only, not vendored |

### Corpus test

`TestCorpus` in `internal/csl` parses external checkouts and fails on any
error or warning. It is skipped unless the directories are given:

```sh
CITEPROC_TEST_SUITE=/path/to/test-suite/processor-tests/humans \
CITEPROC_CSL_STYLES=/path/to/styles \
CITEPROC_CSL_LOCALES=/path/to/locales \
go test ./internal/csl -run TestCorpus -v
```

At the commits above, all 845 test-suite styles, all 10,865 repository
styles (2,863 independent, 8,002 dependent) and all 63 locales parse with no
errors and no warnings.

Without a licence, the CSL test-suite fixtures can't be redistributed, so
they are not committed here. The test runner reads them from a local checkout
of the pinned commit (the path will come from an environment variable; see the
plan for day 3). Fixture names alone, as in the lists below, are fine to
commit. The CSL styles embedded in fixtures are CC BY-SA 3.0 in any case.

## Files

| File | Contents |
|---|---|
| `hayagriva-pass.txt` | The 511 test-suite fixtures hayagriva 0.10.1 passes |
| `apa-subset.txt` | The 263 fixtures whose CSL uses only features that `styles/apa.csl` uses |
| `apa-target.txt` | The 179 fixtures in both lists: the denominator of the day-6 target (pass at least 80%) |

Lines starting with `#` are comments. The ratchet file `pass.txt`
(`plan.md` §4) is added once the runner exists and the first fixtures pass.
