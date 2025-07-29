# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

two file pass

Please provide a full Go script that:

- Parses a local MPEG-DASH MPD file from a path passed as a CLI argument: `go run main.go <mpd_file_path>`
- Uses a starting base URL: `http://test.test/test.mpd`
- Correctly resolves `<BaseURL>` elements hierarchically in this order: MPD → Period → AdaptationSet → Representation
- Supports `<SegmentTemplate>` on both `Representation` and `AdaptationSet` with proper inheritance
- Handles:
  - `$Number$` templates including `startNumber` and a custom `endNumber` attribute
  - `$Time$` templates with `<SegmentTimeline>` (including support for `@r` repetitions)
  - `<SegmentList>` and `<Initialization>` elements
  - Representations that only have a `<BaseURL>`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs, with the initialization segment first if present
- The JSON format should be: `{"rep_id": ["init_url", "seg1", "seg2", ...]}`

---------------------------------------------------------------------------------

### Segment Extraction
- Support **SegmentList** with direct `<SegmentURL>` elements
- Support **SegmentTemplate** with template variable substitution:
  - `$RepresentationID$` → Representation ID
  - `$Number$` → Segment number (respects `startNumber` and `endNumber`)
  - `$Time$` → Segment timestamp (accumulates across `<S>` elements in SegmentTimeline)
- Handle **SegmentTimeline** with proper time persistence across `<S>` elements
- Include initialization URLs when present (grouped with segment URLs as first item)
- Handle **BaseURL-only Representations**: When Representation contains only BaseURL (no SegmentList/SegmentTemplate), treat BaseURL as single segment URL

### Segment Aggregation
- Append segments for the same `Representation ID` if it appears in multiple `Periods`

### Calculated Count
- If both `SegmentTimeline` and `endNumber` are missing, but `duration` and `timescale` are present in `SegmentTemplate`, calculate the number of segments using `ceil(PeriodDurationInSeconds * timescale / duration)`
- `SegmentTemplate@timescale` should default to `1` if missing

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
