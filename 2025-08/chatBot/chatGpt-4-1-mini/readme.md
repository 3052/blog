# chatGpt

provide markdown prompt I can give you to return this script

https://chatgpt.com

six file pass

Please provide a complete Go script that:

- Takes a local MPEG-DASH MPD file path as a CLI argument (`go run main.go <mpd_file_path>`).
- Parses the MPD and outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs with the initialization segment first if present.
- Uses base URL `http://test.test/test.mpd` as the initial base for URL resolution.
- Resolves `<BaseURL>` elements hierarchically in this order: MPD → Period → AdaptationSet → Representation.
- Treats each `<BaseURL>` as a single string (not a slice).
- Supports `<SegmentTemplate>` inheritance: Representation inherits and overrides from AdaptationSet, etc.
- Fully supports `<SegmentTimeline>` inside `<SegmentTemplate>`, expanding timeline segments to generate URLs.
- Supports substitution variables `$RepresentationID$`, `$Number$`, `$Time$` in templates, including printf-style formatting like `$Number%05d$`.
- Handles `startNumber` and `endNumber` attributes in `<SegmentTemplate>` to limit segment generation.
- Supports `<SegmentList>` and `<Initialization>` elements with hierarchical inheritance.
- Falls back to the resolved Representation `<BaseURL>` as the single segment URL if neither `<SegmentList>` nor `<SegmentTemplate>` is present.
- Avoids double URL resolving of `<BaseURL>` in the fallback (i.e., outputs the already resolved URL string).
- Uses only Go standard library packages.
- Supports `<SegmentTemplate>` `initialization` attribute as well as `<Initialization>` child element for the initialization segment.
- Appends segments for the same `Representation@id` if it appears in multiple `<Period>` elements.
- If both `SegmentTimeline` and `endNumber` are missing, and `duration` + `timescale` are present, calculates the number of segments as `ceil(PeriodDurationInSeconds * timescale / duration)`.
- Parses `<Period duration="...">` ISO8601 durations (PT#H#M#S) for segment count calculation.
