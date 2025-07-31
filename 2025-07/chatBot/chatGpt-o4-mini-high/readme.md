# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

one file pass

When I say “generate DASH segment URL resolver”, please reply with a complete
Go program (using only the standard library) that:

* Reads an MPEG-DASH MPD XML file path from the CLI (`go run main.go <mpd_file_path>`)
* Starts with `const defaultBase = "http://test.test/test.mpd"`
* Chains every `<BaseURL>` at the MPD, Period, AdaptationSet, and Representation levels via `net/url.URL.ResolveReference`
* Supports both:

  1. **`<SegmentList>`**

     * `<Initialization sourceURL="…">`
     * Multiple `<SegmentURL media="…">`
  2. **`<SegmentTemplate>`** (at the AdaptationSet *or* Representation level), handling:

     * `initialization` template with `$RepresentationID$`
     * A `<SegmentTimeline>` with `<S t="…" d="…" r="…">` entries and substitutions for `$Number$` and `$Time$` (including repeats)
     * Numeric `startNumber` / `endNumber`
     * If no timeline or `endNumber` but both `duration` and `timescale` are present, compute the segment count as `ceil(PeriodDurationSeconds * timescale / duration)`
* Parses the Period’s `duration` attribute (ISO 8601, e.g. “PT60S”) for fallback counts
* Outputs a JSON map from each `Representation.ID` to its ordered list of fully-resolved segment URLs

Use only Go’s standard library, include error handling, and print the final
JSON to stdout.

---

- Substitutes `$RepresentationID$`, `$Number…$` (with optional zero‐padding), and `$Time…$` placeholders
- Inherits missing templates and defaults `timescale` to 1
- Appends segments across multiple `<Period>`s for the same representation ID
- Parses ISO 8601 durations like `PT2H13M19.040S`
- If a Representation only has its own `<BaseURL>`, returns that URL as the sole segment
