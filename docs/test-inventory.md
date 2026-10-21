# Test inventory: what we port from hayagriva

Status: day-1 deliverable of the spike (`plan.md` §5, day 1). Companion to
`docs/port-map.md`.

This inventory answers three questions:

1. How many CSL test-suite fixtures does hayagriva pass?
2. Which of them use only features APA needs?
3. Which hayagriva unit tests should we port?

Sources are pinned in `testdata/README.md`: hayagriva 0.10.1 (`e51299e`) and
the CSL test suite at `6eefc5b` (2026-04-08).

## 1. Summary

| | Fixtures |
|---|---:|
| CSL test suite, `processor-tests/humans` | 845 (646 citation, 199 bibliography) |
| Skipped by hayagriva's harness | 153 |
| Testable by hayagriva's harness | 692 |
| **Passed by hayagriva** (`testdata/hayagriva-pass.txt`) | **511** (383 citation, 128 bibliography): 74% of testable, 60% of all |
| **APA-feature subset** (`testdata/apa-subset.txt`) | **263** |
| **APA subset ∩ hayagriva passes** (`testdata/apa-target.txt`) | **179**: the day-6 target is to pass at least 80% of these, i.e. 144 |
| hayagriva-local fixtures (`tests/local/*.txt`) | 17, all CSL-JSON, 6 in the APA subset |

All 511 names in hayagriva's pass list still exist in the pinned suite.
hayagriva tracks the suite's `master` branch, so its list could drift from a
newer suite; we pin a commit instead.

**Licence finding.** The test-suite repository has no licence (GitHub reports
none and there is no licence file). We therefore don't vendor fixtures. The
runner reads a checkout of the pinned commit, which is the mitigation already
listed in `plan.md` §6. Fixture *names* are committed in `testdata/`.

## 2. Why hayagriva skips 153 fixtures

These filters are in `tests/citeproc.rs:401-530`. The Go runner starts with the
same named skip reasons so pass counts compare, then lifts them as features
land.

| Skip reason | Fixtures | Of which in the APA subset | Lift in Go? |
|---|---:|---:|---|
| `BIBENTRIES` / `BIBSECTION` / `CITATIONS` section present | 43 | 2 | Later (`CITATIONS` = multi-step document state) |
| HTML in a citation-mode result | 48 | 10 | Yes, once the HTML formatter handles citations |
| Cite item with `prefix`/`suffix`/`position`/`near-note` | 20 | 11 | **Yes: APA needs prefix/suffix** |
| HTML markup or a `note` field in the input | 24 | 8 | Yes: CSL-JSON rich text (`<i>`, `nocase`) |
| Date range in the input | 14 | 2 | Yes: ranges are supported from day 3 |
| Skipped by name (XML parser bugs, plugin) | 4 | 1 | Re-check with `encoding/xml` |
| **Total** | **153** | **34** | |

## 3. The APA-feature subset

### Method

A fixture is in the subset when every CSL feature its style uses also occurs
in `styles/apa.csl`. Features are collected by walking the style XML:

- **Elements** count by name. `<info>` is ignored. Inside `<locale>`, only
  element names are recorded, plus the attributes of `style-options`, `date`
  and `date-part`.
- **Attributes** count by presence. For attributes that select behaviour, the
  value is part of the feature: `form`, `and`, `name-as-sort-order`,
  `delimiter-precedes-last`, `collapse`, `disambiguate-*`,
  `givenname-disambiguation-rule`, `text-case`, `font-*`, `display`, `match`,
  `position`, `sort`, `page-range-format`, `date-parts`, `et-al-use-last`, and
  so on.
- **Variables** count by presence, except the special ones (`citation-number`,
  `year-suffix`, `locator`, `citation-label`,
  `first-reference-note-number`), which count by value.
- **Generic formatting attributes** (affixes, delimiter, font, text case,
  quotes, strip-periods, display) on ordinary elements count independently of
  the element. So `font-variant="small-caps"` on any element is one feature,
  and APA's affixes on `<text>` cover affixes on `<number>`.
- `style@class` counts by presence only. Note-style fixtures whose features
  are otherwise APA's are in the subset; position tests keep most of them out.

APA has 113 features by this definition. The classification is a throwaway
Python script (not committed). The day-3 Go fixture parser will reimplement it
so the lists can be regenerated.

**Limits:**
- The method looks at the CSL only, not at the input data.
- It is strict: one non-APA feature excludes a fixture.
- 133 fixtures that hayagriva passes are exactly one feature away. The
  commonest blockers are:
  - `date@prefix` (12)
  - `name@and=text` (10)
  - `style@default-locale` (9)
  - `givenname-disambiguation-rule=by-cite` (7)
  - `key@sort=descending` (5)
  - `name@form=count` (5)

  These are cheap to support and make good day-6 stretch fixtures.

### Per category

| Category | Total | Testable (hayagriva) | hayagriva passes | APA subset | APA subset ∩ passes |
|---|---:|---:|---:|---:|---:|
| affix | 9 | 2 | 2 | 6 | 1 |
| bugreports | 83 | 55 | 27 | 13 | 7 |
| collapse | 21 | 16 | 11 | 4 | 3 |
| condition | 17 | 17 | 17 | 11 | 11 |
| date | 101 | 88 | 77 | 13 | 11 |
| decorations | 7 | 3 | 1 | 4 | 1 |
| disambiguate | 71 | 61 | 36 | 5 | 4 |
| display | 5 | 4 | 1 | 0 | 0 |
| etal | 4 | 2 | 2 | 0 | 0 |
| flipflop | 19 | 7 | 1 | 15 | 1 |
| form | 3 | 3 | 3 | 3 | 3 |
| fullstyles | 5 | 5 | 3 | 0 | 0 |
| group | 7 | 6 | 5 | 4 | 3 |
| integration | 14 | 2 | 2 | 1 | 0 |
| label | 19 | 16 | 10 | 15 | 10 |
| locale | 23 | 23 | 23 | 11 | 11 |
| locator | 6 | 6 | 4 | 5 | 3 |
| magic | 40 | 36 | 25 | 14 | 8 |
| name | 111 | 102 | 71 | 46 | 37 |
| nameattr | 97 | 97 | 93 | 13 | 13 |
| nameorder | 6 | 6 | 6 | 4 | 4 |
| number | 20 | 14 | 13 | 8 | 7 |
| page | 10 | 10 | 5 | 5 | 2 |
| plural | 7 | 7 | 1 | 0 | 0 |
| position | 16 | 4 | 4 | 0 | 0 |
| punctuation | 16 | 14 | 8 | 6 | 2 |
| quotes | 4 | 4 | 2 | 1 | 1 |
| sort | 54 | 45 | 34 | 20 | 15 |
| substitute | 7 | 7 | 5 | 6 | 5 |
| textcase | 31 | 21 | 12 | 24 | 10 |
| other (namespaces, simplespace, sortseparator, testers, unicode, variables, virtual) | 12 | 9 | 7 | 6 | 6 |
| **Total** | **845** | **692** | **511** | **263** | **179** |

### APA-subset fixtures that hayagriva fails but could test (50)

These fixtures pass hayagriva's skip filters but fail. They are where the Go
port can beat hayagriva, and they line up with the deviations in
`docs/port-map.md` §9:

- **Text case and quotes (12):** `textcase_*` (7), `flipflop_*` (5).
- **Names (12):** `name_*` (8), `magic_NameSuffix*` (2),
  `magic_CapitalizeFirstOccurring*` (2).
- **Plurals and labels (7):** `label_*` (4), `locator_*` (2),
  `number_PlainHyphenOrEnDashAlwaysPlural`.
- **Page ranges (3):** `page_Expand`, `page_ExpandWeirdComposite`,
  `page_NoOption`. APA uses `page-range-format="expanded"`.
- **Punctuation (4):** `punctuation_*`.
- **Sorting, collapse and disambiguation (5):** `sort_NameVariable`,
  `sort_Quotes`, `collapse_AuthorCollapseNoDateSorted`,
  `disambiguate_YearSuffixAndSort`, `bugreports_CollapseFailure`.
- **Other (7):** `bugreports_ApostropheOnParticle`,
  `bugreports_ContainerTitleShort`, `date_InPress`, `date_String`,
  `decorations_NestedQuotes`, `group_SuppressTermWhenNoOutputFromPartialDate`,
  `substitute_SubstituteOnlyOnceTermEmpty`.

### Coverage of APA's key features

The strict subset is narrow for disambiguation, because those fixtures tend to
combine many features. This table uses a looser measure: fixtures whose CSL
uses a given APA setting at all.

| APA feature | Fixtures using it | hayagriva passes |
|---|---:|---:|
| `disambiguate-add-year-suffix="true"` | 118 | 55 |
| `disambiguate-add-names="true"` | 105 | 34 |
| `disambiguate-add-givenname="true"` | 95 | 36 |
| `givenname-disambiguation-rule="primary-name-with-initials"` | **1** | **0** |
| `collapse="year"` | 53 | 28 |
| `<text variable="year-suffix">` | 44 | 23 |
| `et-al-use-last="true"` (bibliography) | 7 | 4 |
| `<key names-min=…>` | 10 | 5 |
| `page-range-format="expanded"` | 22 | 7 |
| `name-as-sort-order="all"` | 140 | 61 |
| `and="symbol"` | 68 | 34 |
| `delimiter-precedes-last="always"` | 90 | 34 |
| `initialize-with` | 154 | 72 |
| `hanging-indent="true"` | 83 | 30 |
| `<date form="text">` | 121 | 78 |
| `<text variable="locator">` | 125 | 40 |
| Cite-item `suppress-author` / `author-only` | **0** | **0** |
| Cite-item `prefix` / `suffix` | 13 / 24 | 0 (skipped) |

**Gaps the APA goldens (`testdata/apa/`) must fill.** The suite gives little
or no evidence for these APA behaviours:

- **Given-name disambiguation with initials.** Only
  `disambiguate_PrimaryNameWithInitialsLimitedToPrimary` uses APA's rule, and
  hayagriva fails it.
- **Narrative citations.** No fixture uses `suppress-author` or `author-only`.
- **Cite-item prefix and suffix.** hayagriva skips every fixture that has
  them.
- **21+ authors.** Only 7 fixtures use bibliography `et-al-use-last`.

## 4. hayagriva-local fixtures (`tests/local/`, 17)

All are in the CSL test-suite format and use CSL-JSON. They are part of
hayagriva (MIT/Apache-2.0), so they can be vendored into
`testdata/hayagriva/` with attribution. Port all of them.

| Fixture | Mode | In APA subset |
|---|---|---|
| `affix_DoiUrlPrefix` | bibliography-full | yes |
| `choose_UseAncestorsDelimiter` | citation | yes |
| `page_NumberPage`, `page_NumberPageMacroSort`, `page_NumberPageSort` | citation | yes |
| `sort_Unicode` | citation | yes |
| `collapse_CitationNumberRangesSeparated`, `collapse_CitationNumberRangesSort` | citation | no (numeric) |
| `date_DateBCAdjusted`, `date_LocalizedDateDelimiters`, `date_NegativeDateSort`, `date_NegativeDateSortViaMacro`, `date_NegativeDateSortViaMacroOnYearMonthOnly` | citation | no |
| `name_PartInheritsFormatting`, `name_SubsequentAuthorSubstitute` | bibliography | no |
| `position_IbidIsNotIbidWithLocator`, `position_IbidWithLocatorIsIbid` | citation | no (note) |

The runner must accept `bibliography-full` mode (bibliography plus links),
and section markers with any number of `=` signs. `sort_Unicode` and
`name_SubsequentAuthorSubstitute` use four.

## 5. hayagriva unit tests

hayagriva has 67 `#[test]` functions. The table lists each one, with the
decision for the Go port.

| File | Tests | Port? | Go destination |
|---|---|---|---|
| `src/types/mod.rs` | `parse_num`, `test_preserve_space_separator` | **Yes** | `internal/value/numeric_test.go` |
| | `test_issue_227` (`page-range: 100` from YAML) | Adapt: CSL-JSON `"page": 100` | `internal/value/pages_test.go` |
| `src/types/numeric.rs` | `test_mixed_range` | **Yes** | `internal/value/numeric_test.go` |
| `src/types/page.rs` | `group_by`, `nonnumeric_page` | **Yes** | `internal/value/pages_test.go` |
| `src/types/persons.rs` | `person_initials`, `person_middle_initial` | **Yes** | `internal/value/name_test.go` |
| | `person_non_dropping_initials` | Adapt: only if the particle heuristic is kept | `internal/value/name_test.go` |
| | `person_name_retrieval_order` | Adapt: keep the parsing/initials assertions, drop `name_first`/`given_first` | `internal/value/name_test.go` |
| `src/types/strings.rs` | `escaped_brace`, `escaped_dollar`, `escaped_backslash`, `normal_backslash` | No: hayagriva `{}`/`$` markup | — |
| `src/types/time.rs` | `test_duration_parse`, `test_duration_range_parse` | No: YAML durations | — |
| `src/lang/mod.rs` | `fold_char_class`, `fold_words`, `title_case_*` (9), `sentence_case*` (6), `online` | **Yes**, adjusting cases that encode behaviour we change (port-map §9 #16) | `internal/render/textcase_test.go` |
| `src/csl/mod.rs` | `low_year_test` | **Yes** | `internal/render/date_test.go` |
| | `test_transparent_locator_ibid` | Adapt: string locators instead of transparent ones | `internal/render` position test |
| | `issue_243` (APA narrative citation, falls back to editors) | **Yes** | `testdata/apa/` golden |
| | `ibid_handling_with_deutsche_sprache_csl` | Adapt: needs that style; its expected `"()"` encodes a quirk | Optional |
| | `test_csl` (smoke test over many styles) | Adapt: smoke test over embedded styles | `citeproc` |
| | `test_chapter_field` | No: hayagriva `Entry` model | — |
| | `test_alphanumeric_disambiguation`, `issue_48` | No: hayagriva's alphanumeric label generator | — |
| `tests/citeproc.rs` | `test_parse_tests`, `test_local_files`, `test_single_file`, `write_passing` | **Yes**, as the runner, `-run` filtering and `-update-pass` | `internal/testsuite` |
| | `purposes` (APA author / narrative / year / full) | **Yes** | `testdata/apa/` golden |
| | `case_folding` (`de-DE` suppresses title case) | **Yes** | `internal/render` or a `testdata/` golden |
| | `access_date`, `no_author` | No: BibLaTeX input | — |
| `tests/archiver.rs` | `no_dupe_id` + generator | No: CBOR archive | — |
| `src/lib.rs`, `src/io.rs`, `src/interop.rs`, `src/selectors/parser.rs` | selectors, YAML round-trips, BibLaTeX conversion, selector parser | No: out of scope | — |

**Totals:**
- **33 port directly**: 7 in `types`, 18 in `lang`, `low_year_test`,
  `issue_243`, 4 harness tests, `purposes` and `case_folding`.
- **3 need small adaptations**: the `types` tests marked "Adapt".
- **3 need larger adaptations**: `test_transparent_locator_ibid`,
  `test_csl`, and optionally the deutsche-sprache test.
- **28 are not ported**: YAML, BibLaTeX, selectors, the CBOR archive, and
  hayagriva's `Entry` model.

## 6. Next steps this inventory feeds

1. **Day 3:** the runner reads fixtures from a checkout of the pinned commit,
   applies hayagriva's skip reasons (§2) by name, and reports
   pass/fail/skip counts. The `types` tests are ported in the same step.
2. **Day 4–5:** APA goldens cover the gaps in §3: primary-name-with-initials,
   narrative citations, cite-item affixes, 21+ authors.
3. **Day 6:** measure against `testdata/apa-target.txt` (target: at least 144
   of 179), then seed the ratchet `testdata/pass.txt` from whatever passes.
