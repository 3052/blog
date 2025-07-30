# claude

provide markdown prompt I can give you in the future to return this script

https://claude.ai

two file pass

Please provide a complete GoLang script that parses MPEG-DASH MPD files and
extracts segment URLs with the following specifications:

## Requirements

### Input Assumptions
- Input is always a local file path (no network requests)
- Command line usage: `go run main.go <mpd_file_path>`

### Output Format
- JSON object mapping Representation IDs to arrays of resolved segment URLs
- Initialization URL (if exists) should be first item in the array
- Format: `{"representation_id": ["init_url", "segment1_url", "segment2_url", ...]}`
- For BaseURL-only representations: `{"representation_id": ["single_segment_url"]}`

### BaseURL Resolution
- Resolve `BaseURL` elements hierarchically: MPD → Period → AdaptationSet → Representation
- Use starting base URL: `http://test.test/test.mpd`
- Handle both absolute and relative URLs properly
- Do NOT double-resolve Representation BaseURLs (the hierarchical resolution already includes them)

### SegmentTemplate Support
- Support `$RepresentationID$`, `$Number$`, and `$Time$` variable substitution
- Handle padding formats (e.g., `$Number%05d$`, `$Time%09d$`)
- Support both SegmentTimeline and duration-based templates
- Respect `startNumber` and `endNumber` attributes
- For SegmentTimeline: `$Time$` value should persist and accumulate across S elements

### Segment Aggregation
- Append segments for the same `Representation ID` if it appears in multiple `Periods`

### Additional Features
- Support SegmentList with Initialization elements
- Handle representations with only BaseURL (no segments)
- Proper error handling for file reading and XML parsing
- Clean, indented JSON output

## Important Implementation Details
- When processing SegmentTimeline, time values should accumulate across S elements unless explicitly reset by a new `t` attribute
- The `endNumber` attribute should limit segment generation in both timeline and duration-based templates
- For duration-based templates without explicit end, generate 10 segments as example
- All URL resolution should handle relative paths, absolute paths, and full URLs correctly
- When a representation has only BaseURL, use the already-resolved baseURL directly without double-resolving

---

### Segment Extraction
- Support **SegmentList** with direct `<SegmentURL>` elements
- Support **SegmentTemplate** with template variable substitution:
  - `$RepresentationID$` → Representation ID
  - `$Number$` → Segment number (respects `startNumber` and `endNumber`)
  - `$Time$` → Segment timestamp (accumulates across `<S>` elements in SegmentTimeline)
- Handle **SegmentTimeline** with proper time persistence across `<S>` elements
- Include initialization URLs when present (grouped with segment URLs as first item)
- Handle **BaseURL-only Representations**: When Representation contains only BaseURL (no SegmentList/SegmentTemplate), treat BaseURL as single segment URL

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
