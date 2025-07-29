# gemini pro

provide markdown prompt I can give you in the future to return this script

https://gemini.google.com

one file pass

Please provide a complete, self-contained Go script in a single `main.go` file that performs the following actions:

## 📜 Script Requirements

1.  **Execution**
    * Parses a local MPEG-DASH MPD file path passed as a command-line argument: `go run main.go <path_to_mpd_file>`.

2.  **URL Resolution**
    * Uses the hardcoded URL `http://test.test/test.mpd` as the top-level base for resolving all relative paths.
    * **Respects `Period@BaseURL`**: If a `<BaseURL>` element exists within a `<Period>`, it must be resolved relative to the main manifest URL. All segments within that `Period` must then be resolved against this new period-specific base URL.

3.  **Segment Generation**
    The script must support two methods for generating segment lists, using the first one if available:
    * **Primary (`<SegmentTimeline>`)**: If a `<SegmentTemplate>` contains a `<SegmentTimeline>`, the script must generate the segment list by iterating through the `<S>` elements, correctly handling their `t` (time), `d` (duration), and `r` (repeat) attributes.
    * **Fallback (Duration-based)**: If no `<SegmentTimeline>` is found, the script must calculate the total number of segments. This is done by dividing the `MPD@mediaPresentationDuration` by the segment duration (derived from `SegmentTemplate@duration` and `SegmentTemplate@timescale`).
    * **URL Templates**: The script must correctly substitute the `$RepresentationID$`, `$Number$`, and `$Time$` placeholders when constructing the final URLs.

4.  **Output**
    * The script must print a single JSON object to standard output.
    * This JSON object should map each `Representation@id` string to a list of fully resolved segment URL strings: `{ "rep_id": ["init_url", "segment_1_url", "segment_2_url", ...] }`.
    * The initialization segment's URL must be the first element in the list if it's defined in the template.

---

- Resolves `<BaseURL>` elements hierarchically: MPD → Period → AdaptationSet → Representation
- Supports `<SegmentTemplate>` on both AdaptationSet and Representation, with inheritance
- Handles:
  - `$RepresentationID$`, `$Number$`, `$Time$` substitutions, including formatted patterns like `$Number%05d$`
  - `startNumber` and `endNumber`
  - `$Time$` using `<SegmentTimeline>` and `@r` repetitions (each `<S>` repeated `1 + r` times)
  - If both `SegmentTimeline` and `endNumber` are missing, and `duration` + `timescale` are present, calculate number of segments as:
    `ceil(PeriodDurationInSeconds * timescale / duration)`
  - Defaults `timescale` to `1` if omitted
- Supports `<SegmentList>` and `<Initialization>` elements
- Falls back to `<BaseURL>` segments if nothing else exists
- Appends segments for the same `Representation@id` if it appears in multiple `<Period>` elements
- Uses **only** `net/url.URL.ResolveReference` for URL resolving (no other logic)
- Prefers `Period@duration` over `MPD@mediaPresentationDuration` for segment count calculations
