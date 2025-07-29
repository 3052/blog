# gemini pro

provide markdown prompt I can give you in the future to return this script

https://gemini.google.com

Please provide the full Go script that:

- Parses a local MPEG-DASH MPD file from a path passed as a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs, with the initialization segment first if present:
  `{ "rep_id": ["init_url", "seg1", "seg2", ...] }`

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
