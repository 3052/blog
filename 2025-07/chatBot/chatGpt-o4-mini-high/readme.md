# chatGpt

provide markdown prompt I can give you in the future to return this script, do
not wrap in a code block

https://chatgpt.com

two file pass

Provide a complete Go program (using only the standard library) that:

1. Reads an MPEG-DASH MPD XML file path from the CLI (`go run main.go <mpd_file_path>`)
2. Starts with `const defaultBase = "http://test.test/test.mpd"`
3. Chains every `<BaseURL>` at the MPD, Period, AdaptationSet, and Representation levels via `net/url.URL.ResolveReference`
4. Supports both:
   * **`<SegmentList>`**
     * `<Initialization sourceURL="…">`
     * Multiple `<SegmentURL media="…">`
   * **`<SegmentTemplate>`** (at AdaptationSet **or** Representation level), handling:
     * `initialization` template with `$RepresentationID$`
     * A `<SegmentTimeline>` with `<S t="…" d="…" r="…">` entries and substitutions for `$Number$` and `$Time$` (including repeats)
     * Numeric `startNumber` / `endNumber`
     * If no timeline or `endNumber` but both `duration` and `timescale` are present, compute the segment count as `ceil(PeriodDurationSeconds * timescale / duration)`
5. Parses the Period’s `duration` attribute (ISO 8601, e.g. `PT60S`) for fallback counts
6. If a Representation has neither `<SegmentList>` nor `<SegmentTemplate>`, returns its own `<BaseURL>` as the sole segment URL
7. Outputs a JSON map from each `Representation.ID` to its ordered list of fully-resolved segment URLs

Include full error handling and print the final JSON to stdout.

---

- Substitutes `$RepresentationID$`, `$Number…$` (with optional zero‐padding), and `$Time…$` placeholders
- Inherits missing templates and defaults `timescale` to 1
- Appends segments across multiple `<Period>`s for the same representation ID
- Parses ISO 8601 durations like `PT2H13M19.040S`
