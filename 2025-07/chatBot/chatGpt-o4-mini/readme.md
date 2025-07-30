# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

two file pass

Return a complete Go program (using only the standard library) that:

* Reads an MPEG-DASH MPD XML file path from the CLI:
  `go run main.go <mpd_file_path>`
* Starts with
  `const defaultBase = "http://test.test/test.mpd"`
  then chains each `<BaseURL>` element in the hierarchy (MPD → Period → AdaptationSet → Representation) by resolving it with `net/url.URL.ResolveReference`.
* Supports both `<SegmentList>` (with its `<Initialization>` element and `<SegmentURL>` entries) and `<SegmentTemplate>` (with an `initialization` attribute plus either `<SegmentTimeline>` or numeric ranges), inheriting at each level when missing.
* Substitutes placeholders `$RepresentationID[…]$`, `$Number[…]$`, and `$Time[…]$` (including formats like `%05d`) with one regex.
* Persists `Time` across timeline entries and respects `endNumber` for numeric templates.
* Falls back to the fully-resolved Representation base URL itself if no segments are produced.
* Outputs a JSON map from each `Representation@id` to its ordered list of fully-resolved segment URLs (initialization first).

---

- Supports:
  - `<SegmentTimeline>` with proper handling of `@t`, `@d`, `@r`
  - `<SegmentList>` and `<Initialization>` elements
  - Appends segments across multiple `<Period>`s for the same `Representation@id`
  - Defaults `timescale=1` if not present
  - If both `SegmentTimeline` and `endNumber` are missing, and both `duration` and `timescale` are present, calculates number of segments as:
    ```
    ceil(PeriodDurationInSeconds * timescale / duration)
    ```
