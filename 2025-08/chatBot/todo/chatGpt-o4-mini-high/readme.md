# chatGpt

provide markdown prompt I can give you in the future to return this script, do
not wrap in a code block

https://chatgpt.com

six file pass

Please provide a complete Go program (main.go) using only the standard library that:

1. Reads an MPEG-DASH MPD XML file path from the CLI (`go run main.go <mpd_file_path>`)
2. Starts with `const defaultBase = "http://test.test/test.mpd"`
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
8. Appends all segments and outputs a JSON map from each `Representation.ID` to its ordered list of fully-resolved segment URLs, with full error handling and printing to stdout
