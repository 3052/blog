# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

three file pass

Please provide the full Go script that:

- Parses a local MPEG-DASH MPD file from a path passed as a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Resolves `<BaseURL>` elements hierarchically: MPD → Period → AdaptationSet → Representation
- Supports `<SegmentTemplate>` on both AdaptationSet and Representation, with inheritance
- Handles:
  - `$RepresentationID$`, `$Number$`, `$Time$` substitutions
  - `startNumber` and `endNumber`
  - `$Time$` using `<SegmentTimeline>` and `@r` repetitions
  - If both `SegmentTimeline` and `endNumber` are missing, and `duration` + `timescale` are present, calculate number of segments as:
    `ceil(PeriodDurationInSeconds * timescale / duration)`
  - Defaults `timescale` to `1` if omitted
- Supports `<SegmentList>` and `<Initialization>` elements
- Falls back to `<BaseURL>` segments if nothing else exists
- Appends segments for the same `Representation@id` if it appears in multiple `<Period>` elements
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs, with the initialization segment first if present:
  `{ "rep_id": ["init_url", "seg1", "seg2", ...] }`

---------------------------------------------------------------------------------

### Segment Extraction
- Support **SegmentList** with direct `<SegmentURL>` elements
- Handle **SegmentTimeline** with proper time persistence across `<S>` elements
- Include initialization URLs when present (grouped with segment URLs as first item)
- Handle **BaseURL-only Representations**: When Representation contains only BaseURL (no SegmentList/SegmentTemplate), treat BaseURL as single segment URL


### XML Structure Handling
- Properly handle nested `<Initialization sourceURL="..."/>` elements in SegmentList
- Support formatted number patterns like `$Number%05d$` in templates
- Handle template inheritance from AdaptationSet to Representation level
- Parse ISO 8601 duration formats (PT30.5S, PT1M30S, PT1H30M, etc.)

### Key Implementation Details
- Time values in SegmentTimeline must persist and accumulate across `<S>` elements
- When `t` attribute is present in `<S>` element, use it as starting time
- EndNumber takes priority over duration-based segment count calculations
- Template variables should be resolved at the appropriate hierarchy level
- Proper error handling and validation throughout
