# chatGpt

provide markdown prompt I can give you to return this script, do not wrap in a
code block

https://chatgpt.com

two file pass

Write a single, complete Go program named **main.go** that uses **only the
standard library** and does the following. Return **only** the code in one
fenced Go code block (no commentary before or after).

Behavior & CLI

* Program name: `main.go`
* Usage: `go run main.go <mpd_file_path>`
* Read the local MPEG-DASH MPD XML from the path provided on the CLI.
* On success, print to **stdout** a JSON object mapping each `Representation@id` to an **ordered list** of fully resolved segment URLs.
* On any error, print a clear message to **stderr** and exit with a non-zero status.

Constant requirement

* Define at top-level: `const defaultBase = "http://test.test/test.mpd"`

DASH handling scope

* Parse enough of the DASH MPD to:

  * Resolve **BaseURL** inheritance across MPD → Period → AdaptationSet → Representation, falling back to `defaultBase` when needed.
  * Support **SegmentList** (Initialization + `SegmentURL@media` in order).
  * Support **SegmentTemplate**:

    * `initialization` and `media` templates with tokens `$RepresentationID$`, `$Bandwidth$`, `$Number$`, `$Time$`.
    * Literal `$$` in templates must be unescaped to `$`.
    * `timescale`, `duration`, `startNumber` (with sane defaults), and `presentationTimeOffset` field presence handled.
    * **SegmentTimeline** with repeated `S` elements; handle `R >= 0` and `R = -1` (“repeat to end of Period/MPD duration”). If `R = -1` but no total duration is known, return an error.
    * If `media` contains `$Time$` but there's **no** `SegmentTimeline`, return an error.
    * For number-based expansion (no `SegmentTimeline`), compute count from Period or MPD duration using `duration/timescale`; error if the duration is unknown or invalid.
* If neither `SegmentList` nor `SegmentTemplate` is available, but a `Representation` has a `BaseURL` pointing to a single resource, include that single resolved URL; otherwise error.
* Maintain **segment order** exactly as defined by the MPD constructs.

Durations

* Implement a minimal ISO-8601 duration parser for forms like `PT#H#M#S` and `P#DT#H#M#S` (seconds may be fractional). If parsing fails, log a warning and treat as unknown duration.
* Provide a helper `hmsToDuration(daysStr, hoursStr, minsStr, secsStr string) time.Duration`.
* Ensure the fallback call in the PT-only path passes `""` (empty string) for days (not `0`) to avoid type errors.

URL resolution

* Use `net/url` to resolve relative URLs with `ResolveReference`, layering BaseURL values in order (MPD → Period → AdaptationSet → Representation). Always start from `defaultBase`.

Output

* Encode the final map as pretty JSON to **stdout**.
* Use `log` for diagnostics to **stderr** (no timestamps).

Constraints

* **Standard library only.**
* Clean, idiomatic, and compilable Go.
* Include all necessary structs and XML tags to decode the MPD parts described above.
* Provide robust error messages that mention the failing context (e.g., which representation or timeline entry failed).

---

3. Chains every `<BaseURL>` at the MPD, Period, AdaptationSet, and Representation levels via `net/url.URL.ResolveReference`
4. Supports both:
   * **`<SegmentList>`** with `<Initialization sourceURL="…">` and multiple `<SegmentURL media="…">`
   * **`<SegmentTemplate>`** (at AdaptationSet or Representation) handling:
     * `initialization` template with `$RepresentationID$`
     * `<SegmentTimeline>` entries `<S t="…" d="…" r="…">` and substitutions for `$Number$`, `$Time$` (including repeats)
     * Numeric `startNumber`/`endNumber`
     * Fallback when no timeline or `endNumber` but `duration`+`timescale` present, computing count via `ceil(PeriodDurationSeconds * timescale / duration)`
     * Support for formatted placeholders like `$Number%08d$` and `$Time%05d$`
5. Parses the Period’s ISO-8601 `duration` attribute (e.g. `PT60S`) for fallback counts
6. If a Representation lacks both SegmentList and SegmentTemplate, returns its own `<BaseURL>` as the sole segment URL
7. Accumulates segments across multiple `<Period>`s for the same Representation ID
