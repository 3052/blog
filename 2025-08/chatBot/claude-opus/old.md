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
- **MUST use only `net/url.URL.ResolveReference` for URL resolution, no other package or logic**

### SegmentTemplate Support
- Respect `startNumber` and `endNumber` attributes
- Support `$RepresentationID$`, `$Number$`, and `$Time$` variable substitution
- Handle padding formats (e.g., `$Number%05d$`, `$Time%09d$`)
- Support both SegmentTimeline and duration-based templates
- For SegmentTimeline: `$Time$` value should persist and accumulate across S elements
- `SegmentTemplate@timescale` should default to `1` if missing

### Calculated Count
- If both `SegmentTimeline` and `endNumber` are missing, but `duration` and `timescale` are present in `SegmentTemplate`, calculate the number of segments using `ceil(PeriodDurationInSeconds * timescale / duration)`
- Parse Period duration from ISO 8601 format (e.g., "PT634.566S", "PT10M30S")

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
- For duration-based templates without explicit end and without period duration, generate 10 segments as example
- All URL resolution should handle relative paths, absolute paths, and full URLs correctly using only `net/url.URL.ResolveReference`
- When a representation has only BaseURL, use the already-resolved baseURL directly without double-resolving
