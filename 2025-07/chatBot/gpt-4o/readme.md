# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

two file pass

Please provide the **full Go script** (standard library only) that:

- Parses a local MPEG-DASH MPD file from a path passed as a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs, with the initialization segment first if present
- Supports:
  - `<SegmentTemplate>` on both `AdaptationSet` and `Representation`, with inheritance
  - `$RepresentationID$`, `$Number$`, `$Time$` substitutions including formatted forms like `$Number%05d$`
  - `<SegmentTimeline>` including correct handling of `@t`, `@d`, and `@r`
  - `<SegmentList>` and `<Initialization>` segment references
  - Resolves `<BaseURL>` hierarchically: MPD → Period → AdaptationSet → Representation
  - Falls back to `<BaseURL>` segment URLs when no other segment info is present
  - Uses **only** `net/url.URL.ResolveReference` for all URL resolution (no manual path logic)

---

- Handles:
  - If both `SegmentTimeline` and `endNumber` are missing, and `duration` + `timescale` are present, calculate number of segments as:
    `ceil(PeriodDurationInSeconds * timescale / duration)`
  - Defaults `timescale` to `1` if omitted
- Appends segments for the same `Representation@id` if it appears in multiple `<Period>` elements
- Prefers `Period@duration` over `MPD@mediaPresentationDuration` for segment count calculations
