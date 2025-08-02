# claude

provide prompt I can give you to return this script

https://claude.ai

one file pass

Please provide a complete GoLang script that parses MPEG-DASH MPD files and
extracts segment URLs with the following specifications:

### Input Assumptions
- Input is always a local file path (no network requests)
- Command line usage: `go run main.go <mpd_file_path>`

### BaseURL Resolution
- Resolve `BaseURL` elements hierarchically: MPD → Period → AdaptationSet → Representation
- Use starting base URL: `http://test.test/test.mpd`

### SegmentTemplate Support
- Support SegmentTemplate at both AdaptationSet and Representation levels
- Representation-level templates inherit and override AdaptationSet-level templates
- Support template variable substitution:
  - `$RepresentationID$` → Representation ID
  - `$Number$` → Segment number (with formatting like `$Number%05d$`)
  - `$Time$` → Segment timestamp (accumulates across `<S>` elements in SegmentTimeline)
- Respect `startNumber` and `endNumber` attributes to control segment generation range
- When SegmentTimeline exists, use it for precise segment generation (endNumber ignored)
- When no SegmentTimeline, generate segments from startNumber to endNumber

### Segment Types Support
- SegmentBase (single file with initialization)
- SegmentList (explicit list of segments)
- SegmentTemplate (template-based URL generation)

### Output Format
- JSON object mapping Representation IDs to arrays of resolved segment URLs
- Format: `{"representation_id": ["init_url", "segment1_url", "segment2_url", ...]}`

### Timeline Processing
- Process `<S>` elements with proper time accumulation using duration (`d` attribute)
- Handle absolute timestamps (`t` attribute) when present
- Support repeat counts (`r` attribute) for repeated segments
