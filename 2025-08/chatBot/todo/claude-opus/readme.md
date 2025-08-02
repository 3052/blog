# claude

provide prompt I can give you to return this script

https://claude.ai

six file pass

Please provide a complete GoLang script that parses MPEG-DASH MPD files and
extracts segment URLs with the following specifications:

### Input Assumptions
- Input is always a local file path (no network requests)
- Command line usage: `go run main.go <mpd_file_path>`

### Output Format
- JSON object mapping Representation IDs to arrays of resolved segment URLs
- Format: `{"representation_id": ["init_url", "segment1_url", "segment2_url", ...]}`

### BaseURL Resolution
- Resolve `BaseURL` elements hierarchically: MPD → Period → Representation
- Use starting base URL: `http://test.test/test.mpd`
- BaseURL is a string type, not a slice
- When a representation has only BaseURL and no segment information, use the already-resolved baseURL directly

### SegmentTemplate Support
- Respect `startNumber` and `endNumber` attributes
- Support both timeline-based and duration-based templates
- Replace template variables: `$RepresentationID$`, `$Number$`, `$Time$`, `$Bandwidth$`
- Support padded number format `$Number%0Xd$`
- If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
   `timescale` are present, calculate the number of segments using
   `ceil(PeriodDurationInSeconds * timescale / duration)`
- Default `timescale` to `1` if missing

### Additional Requirements
- Support SegmentList with initialization and segment URLs
- Support SegmentBase for single-segment representations
- Properly handle URL resolution for relative and absolute paths
- Include proper error handling for file reading and XML parsing
- Append segments for the same Representation ID if it appears in multiple Periods
- Parse ISO 8601 duration format for Period duration (e.g., PT30S, PT1H30M)
