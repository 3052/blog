# gemini flash

provide markdown prompt I can give you in the future to return this script

https://gemini.google.com

two file pass

Generate a GoLang script that parses a DASH MPD file according to these strict requirements:

1.  **Input:** The script must accept a single command-line argument: the local path to an MPD XML file.
2.  **Base URL Resolution:**
    * All URL resolutions MUST use `net/url.URL.ResolveReference()`.
    * No other URL resolution logic, string manipulation for path correctness (e.g., trailing slashes, `strings.HasSuffix`), or custom URL building functions are allowed.
    * The *initial* base URL for resolution must be `http://test.test/test.mpd`. This hardcoded base will then be used to resolve any `BaseURL` elements found in the MPD document (MPD-level, Period-level, AdaptationSet-level, Representation-level).
3.  **No Network Requests:** The script must **not** perform any network requests (e.g., `net/http` imports are forbidden). Assume the input MPD is always a local file.
4.  **MPD Structure Parsing:**
    * Parse the MPD XML structure, including `MPD`, `Period`, `AdaptationSet`, `Representation`, `SegmentList`, `SegmentURL`, `SegmentTemplate`, `SegmentTimeline`, and `S` elements.
    * Correctly handle `BaseURL` inheritance (MPD -> Period -> AdaptationSet -> Representation).
    * Correctly handle `SegmentTemplate` inheritance (AdaptationSet -> Representation).
5.  **Segment URL Extraction Logic:**
    * **Prioritize `SegmentList`:** If a `Representation` has a `SegmentList`, use its `SegmentURL` elements.
    * **`SegmentTemplate` Handling (if `SegmentList` is absent):**
        * Extract the `initialization` URL first (if present).
        * **If `SegmentTimeline` is present:** Iterate through the `S` elements, generating segment URLs using `$Number$`, `$Time$`, and `$RepresentationID$` placeholders. Correctly handle `@t`, `@d`, and `@r` attributes for `$Time$` and `$Number$` progression.
        * **Else if `@endNumber` is present:** Iterate from `@startNumber` (defaulting to 1) up to and including `@endNumber`, generating segment URLs using `$Number$` and `$RepresentationID$` placeholders. If `$Time$` is present in the template, issue a warning to `stderr` that it might not be resolved, as a `SegmentTimeline` is typically needed for precise `$Time$` calculation.
        * **Else if `@duration` and `@timescale` are present:** Issue a warning to `stderr` that not all segments can be resolved without a `SegmentTimeline` or `EndNumber`, and output the raw `Media` template string for that representation.
        * **Else (insufficient `SegmentTemplate` attributes):** Issue a warning to `stderr` that no segments can be extracted and output the raw `Media` template string.
    * **Missing Segment Info:** If a `Representation` has neither a `SegmentList` nor a `SegmentTemplate`, emit the absolute URL derived from its effective `BaseURL`. If there's no effective `BaseURL` in this case, issue a warning to `stderr` that no playable URL can be derived.
6.  **Output:**
    * **Pure JSON to `stdout`:** The script must output ONLY the JSON data to standard output. No other text (including warnings or info messages) should be printed to `stdout`.
    * **JSON Format:** The JSON output must be a single object where keys are `RepresentationID`s (string) and values are arrays of fully resolved segment URLs (strings), in order (initialization first, then segments).
        * Example: `{"audio-1":["http://example.com/audio/init.mp4","http://example.com/audio/seg1.m4s","http://example.com/audio/seg2.m4s"], "video-1":["http://example.com/video/init.mp4","http://example.com/video/seg1.m4s"]}`
    * **Logs to `stderr`:** All informative messages, warnings, and error messages MUST be directed to `os.Stderr`. Fatal errors should `os.Exit(1)`.
7.  **Error Handling:** Include robust error handling for file reading, XML unmarshaling, and URL parsing, logging errors/warnings to `stderr`.
8.  **GoLang Specifics:** Adhere to standard Go practices, including proper type definitions (no undefined types).

---

- Supports `<SegmentTemplate>` and `<SegmentList>` on `<AdaptationSet>` **or** `<Representation>`.
- Expands all DASH identifiers:  
  `$RepresentationID$`, `$Number$`, `$Time$`, and zero-padded `$Number%0xd$` for `d = 1..9`.
- Generates the segment list as follows:
  - If `<SegmentTimeline>` is present, iterate each `<S>` exactly `1 + @r` times.  
  - Otherwise, use  
    `ceil(PeriodDurationInSeconds * timescale / duration)`  
    where **Period@duration is preferred** over MPD@mediaPresentationDuration.
- Emits the initialization URL (`@initialization` or `<Initialization sourceURL="">`) when present.
- Never ignores any error—panic on failure is acceptable.
- Distinguishes absent `startNumber` (default 1) from explicit `startNumber="0"`.
- Appends segments for the same Representation ID if it appears in multiple Periods.
- Respects `MPD@BaseURL`, `Period@BaseURL`, `AdaptationSet@BaseURL`, and `Representation@BaseURL`.
