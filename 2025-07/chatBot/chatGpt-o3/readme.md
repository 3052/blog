# chatGpt

provide prompt I can give you to return this script

https://chatgpt.com

two file pass

Write a single, fully-compilable Go program named main.go that uses only the
standard library and satisfies *all* of the requirements below.

### CLI & basic behaviour
* Usage: `go run main.go <mpd_file_path>`
* Read the local MPEG-DASH MPD XML from the given path.
* On success, print to **stdout** a JSON object mapping each `Representation@id` to an **ordered list** of fully-resolved segment URLs.
* On any error, print a clear message to **stderr** via the `log` package, disable timestamps with `log.SetFlags(0)`, and exit non-zero.

### URL resolution & BaseURL inheritance
* **Start the resolution chain with the absolute URL `http://test.test/test.mpd`.**
* Apply any `BaseURL` elements on top of that in this exact order, using `net/url.ResolveReference` each time:
  1. MPD
  2. Period
  3. AdaptationSet
  4. Representation
* If a Representation supplies a `BaseURL` that **doesn’t** end with “/” *and* there is no `Segment*` info at any level, output just that single fully-resolved URL for the representation.

### Segment generation
* Support **SegmentList** and **SegmentTemplate**.
* **SegmentTemplate rules**
  * Accept `<SegmentTimeline>` *or* the attribute pair `startNumber` / `endNumber`.  
    • When only those numbers are present, generate sequentially from `startNumber` (default = 1) **through `endNumber`, inclusive**.  
    • If `endNumber` is present *and* a `<SegmentTimeline>` exists, stop once `Number` exceeds `endNumber`.
  * Always output the `initialization` URL first (if present), then media segments.
* **Token expansion** (with optional `%0Nd` zero-padding) must work in both `media` and `initialization` templates and inside `SegmentURL@media` or `@sourceURL`:  
  `$RepresentationID$`, `$Number$`, `$Time$`.

### Period ordering & segment appending
* If the same `Representation@id` appears in multiple **Periods**, append the segments in Period order.

### Fallback segment count
* If **both** `SegmentTimeline` and `endNumber` are missing **and** both `duration` and `timescale` are present, calculate the number of segments as  
  `ceil(PeriodDurationInSeconds * timescale / duration)`.

### Miscellaneous
* Keep everything in one file (**main.go**) and rely solely on the standard library—no third-party packages, no `go:embed`.
* Use clear, idiomatic Go; comment only where it truly clarifies.
* Output nothing except the complete source code.

## prompts

**Implementation constraints & robustness**

* **Standard library only.**
* Include all necessary structs and XML tags to decode the parts described above.
* Provide robust error messages that mention the failing context (e.g., which representation or timeline entry failed).
* Use `log` for diagnostics to stderr (no timestamps).

**Important implementation details to include**

* A helper for composing the effective BaseURL chain, returning whether the Representation’s BaseURL is a single resource and its resolved URL.
* Template replacement that:

  * Recognizes `$RepresentationID$`, `$Bandwidth$`, `$Number$`, `$Time$` with optional printf-style format specifiers inside the `$…$`.
  * Protects `$$` during token scanning and unescapes afterwards.
* **`durationToUnits` must return a `float64`** computed as `(timescale *
   durationSeconds)` using nanoseconds and floating-point math. When calculating
   counts, use `math.Ceil(totalUnits / float64(durationUnits))`. When
   comparing timeline times against the total duration bound, compare using the
   float bound derived from `durationToUnits`.

**DASH handling scope**

* Parse enough of the MPD to support:

  * **SegmentList**: include `Initialization@sourceURL` first (if present), then each `SegmentURL@media` in order.
  * **SegmentTemplate**:
    * Support **formatted placeholders** like `$Number%08d$`, `$Time%010d$`. For `$RepresentationID$`, allow string verbs (e.g., `%s`, `%q`, `%v`) if present; otherwise use the raw string.
    * Literal `$$` in templates must unescape to a single `$`.
    * Handle attributes: `timescale`, `duration`, `startNumber`, `endNumber` (inclusive), `presentationTimeOffset`.
      * **startNumber semantics**: if the `startNumber` attribute is **missing**, default to `1`; if it is present with value `"0"`, use `0`. (Missing ≠ `"0"`.)
      * **endNumber semantics**: when present and > 0, treat as an **inclusive** upper bound on `$Number$`.
    * **SegmentTimeline** with repeated `S` elements; handle `R >= 0` and `R = -1` (“repeat to end of Period/MPD duration”).
      * If `R = -1` but no total duration is known, return an error.
      * Apply `presentationTimeOffset` (PTO) when computing `$Time$` start positions.
    * If `media` contains `$Time$` but there is **no** `SegmentTimeline`, return an error.
      * Error if total duration is unknown or invalid, or if `duration` ≤ 0.

* If neither `SegmentList` nor `SegmentTemplate` is available and the Representation has **no** single-resource BaseURL, error.

**Durations**

* Implement a minimal ISO-8601 duration parser supporting `PT#H#M#S` and `P#DT#H#M#S` (seconds may be fractional).
* Provide helper: `hmsToDuration(daysStr, hoursStr, minsStr, secsStr string) time.Duration`.
* If parsing fails, **log a warning** to stderr and treat duration as unknown.
* In the PT-only path, ensure the fallback call **passes `""` for days** (not `"0"`).

**URL resolution**

* Use `net/url` to resolve relative URLs with `ResolveReference`, layering
* BaseURL values in order (MPD → Period → AdaptationSet → Representation).
* Always start from `defaultBase`.
