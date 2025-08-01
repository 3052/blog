# chatGpt

provide markdown prompt I can give you to return this script

https://chatgpt.com

one file pass

Please provide a complete Go script that takes a local MPEG-DASH MPD file path
as a CLI argument (`go run main.go <mpd_file_path>`), parses the MPD, and
outputs a JSON object mapping each `Representation@id` to a list of fully
resolved segment URLs with the initialization segment first if present.

Requirements:
- Use base URL `http://test.test/test.mpd` as the initial base for URL resolution.
- Resolve `<BaseURL>` elements hierarchically in this order: MPD → Period → AdaptationSet → Representation.
- Support `<SegmentTemplate>` inheritance: Representation inherits and overrides from AdaptationSet.
- Fully support `<SegmentTimeline>` inside `<SegmentTemplate>`, expanding timeline segments to generate URLs.
- Support substitution variables `$RepresentationID$`, `$Number$`, `$Time$` in templates, including printf-style formatting like `$Number%05d$`.
- Handle `startNumber` and `endNumber` attributes in `<SegmentTemplate>` to limit segment generation.
- Output JSON with keys = representation IDs and values = arrays of fully resolved segment URLs.

Use only Go standard library packages.

## prompts

- Handles:
  - `$Time$` using `<SegmentTimeline>` and `@r` repetitions (each `<S>` repeated `1 + r` times)
  - If both `SegmentTimeline` and `endNumber` are missing, and `duration` + `timescale` are present, calculate number of segments as:
    `ceil(PeriodDurationInSeconds * timescale / duration)`
  - Defaults `timescale` to `1` if omitted
- Supports `<SegmentList>` and `<Initialization>` elements
- Falls back to `<BaseURL>` segments if nothing else exists
- Appends segments for the same `Representation@id` if it appears in multiple `<Period>` elements
- Uses **only** `net/url.URL.ResolveReference` for URL resolving (no other logic)
- Prefers `Period@duration` over `MPD@mediaPresentationDuration` for segment count calculations
