# chatGpt

provide prompt I can give you to return this script

https://chatgpt.com

six file pass

Write a single, complete Go program named **main.go** that uses **only the
standard library** and exactly implements the spec below. 

**Program behavior & CLI**

* Program name: `main.go`
* Usage: `go run main.go <mpd_file_path>`
* Read the local MPEG-DASH MPD XML from the provided CLI path.
* On success, print to **stdout** a JSON object mapping each `Representation@id` to an **ordered list** of fully resolved segment URLs.
* On any error, print a clear message to **stderr** using `log` and exit non-zero. Disable log timestamps (`log.SetFlags(0)`).

**Constant requirement**

* Define at top level: `const defaultBase = "http://test.test/test.mpd"`

**DASH handling scope**

* Parse enough of the MPD to support:

  * **BaseURL inheritance** across MPD → Period → AdaptationSet → Representation, layering with `net/url.ResolveReference`, always starting from `defaultBase`. If a Representation has a `BaseURL` that is a single resource (does **not** end with `/`) and no Segment\* info, include that single resolved URL.
  * **SegmentList**: include `Initialization@sourceURL` first (if present), then each `SegmentURL@media` in order.
  * **SegmentTemplate**:
    * Support `initialization` and `media` templates with tokens `$RepresentationID$`, `$Bandwidth$`, `$Number$`, `$Time$`.
    * Support **formatted placeholders** like `$Number%08d$`, `$Time%010d$`. For `$RepresentationID$`, allow string verbs (e.g., `%s`, `%q`, `%v`) if present; otherwise use the raw string.
    * Literal `$$` in templates must unescape to a single `$`.
    * Handle attributes: `timescale`, `duration`, `startNumber`, `endNumber` (inclusive), `presentationTimeOffset`.
      * **startNumber semantics**: if the `startNumber` attribute is **missing**, default to `1`; if it is present with value `"0"`, use `0`. (Missing ≠ `"0"`.)
      * **endNumber semantics**: when present and > 0, treat as an **inclusive** upper bound on `$Number$`.
    * **SegmentTimeline** with repeated `S` elements; handle `R >= 0` and `R = -1` (“repeat to end of Period/MPD duration”).
      * If `R = -1` but no total duration is known, return an error.
      * Apply `presentationTimeOffset` (PTO) when computing `$Time$` start positions.
    * If `media` contains `$Time$` but there is **no** `SegmentTimeline`, return an error.
    * **Number-based expansion (no SegmentTimeline)**:
      * If `endNumber` is present, generate from `startNumber` through `endNumber` **inclusive**.
      * Otherwise, compute the segment **count = ceil(totalUnits / durationUnits)** where:
        * `totalUnits = durationToUnits(Period/MPD total duration, timescale)` (see “Durations” below, must be float),
        * `durationUnits = SegmentTemplate@duration` (integer units).
      * Error if total duration is unknown or invalid, or if `duration` ≤ 0.

* If neither `SegmentList` nor `SegmentTemplate` is available and the Representation has **no** single-resource BaseURL, error.

* Maintain **segment order** exactly as defined by the MPD constructs. If the same `Representation@id` appears in multiple Periods, append segments in Period order.

**Durations**

* Implement a minimal ISO-8601 duration parser supporting `PT#H#M#S` and `P#DT#H#M#S` (seconds may be fractional).
* Provide helper: `hmsToDuration(daysStr, hoursStr, minsStr, secsStr string) time.Duration`.
* If parsing fails, **log a warning** to stderr and treat duration as unknown.
* In the PT-only path, ensure the fallback call **passes `""` for days** (not `"0"`).

**URL resolution**

* Use `net/url` to resolve relative URLs with `ResolveReference`, layering BaseURL values in order (MPD → Period → AdaptationSet → Representation). Always start from `defaultBase`.

**Output**

* Encode the final map as pretty JSON to **stdout**.

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
* **`durationToUnits` must return a `float64`** computed as `(timescale * durationSeconds)` using nanoseconds and floating-point math. When calculating counts, use **`math.Ceil(totalUnits / float64(durationUnits))`**. When comparing timeline times against the total duration bound, compare using the float bound derived from `durationToUnits`.
