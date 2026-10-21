# testdata

Test fixtures and test lists for citeproc. See `docs/test-inventory.md` for
how the lists were produced.

## Pinned sources

| Source | Commit | Licence | Use here |
|---|---|---|---|
| [hayagriva](https://github.com/typst/hayagriva) 0.10.1 | `e51299ecd1a9f62b2f069af2f554ae4e84d50527` | MIT OR Apache-2.0 | Pass list, local fixtures and unit tests are ported from it |
| [CSL test suite](https://github.com/citation-style-language/test-suite) | `6eefc5b07c6969ab8999e48542acbcc131cba864` (2026-04-08) | **None stated** (no licence file; GitHub reports no licence) | Fetched at this commit for test runs, **not vendored** |

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
