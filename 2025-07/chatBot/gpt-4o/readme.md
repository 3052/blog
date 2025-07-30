# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

one file pass

Please provide the full Go script (standard library only) that:

- Parses a local MPEG-DASH MPD file from a path passed as a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs, with the initialization segment first if present
- Supports:
  - `<SegmentTemplate>` on both `AdaptationSet` and `Representation`, with inheritance
  - `$RepresentationID$`, `$Number$`, `$Time$` substitutions including formatted forms like `$Number%05d$`
  - `<SegmentTimeline>` including correct handling of `@t`, `@d`, and `@r`
  - Resolves `<BaseURL>` hierarchically: MPD → Period → AdaptationSet → Representation
  - Honors `startNumber` and `endNumber` from `SegmentTemplate`

---

- Handles:
  - If both `SegmentTimeline` and `endNumber` are missing, and `duration` + `timescale` are present, calculate number of segments as:
    `ceil(PeriodDurationInSeconds * timescale / duration)`
  - Defaults `timescale` to `1` if omitted
- Supports `<SegmentList>` and `<Initialization>` elements
- Falls back to `<BaseURL>` segments if nothing else exists
- Appends segments for the same `Representation@id` if it appears in multiple `<Period>` elements
- Uses **only** `net/url.URL.ResolveReference` for URL resolving (no other logic)
- Prefers `Period@duration` over `MPD@mediaPresentationDuration` for segment count calculations
