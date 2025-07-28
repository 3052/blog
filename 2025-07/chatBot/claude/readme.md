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

### BaseURL Resolution
- Resolve `BaseURL` elements hierarchically: MPD → Period → AdaptationSet → Representation
- Use starting base URL: `http://test.test/test.mpd`
- Handle both absolute and relative URLs properly
- Avoid double resolution when Representation contains only BaseURL

### Segment Extraction
- Support **SegmentList** with direct `<SegmentURL>` elements
- Support **SegmentTemplate** with template variable substitution:
  - `$RepresentationID$` → Representation ID
  - `$Number$` → Segment number (respects `startNumber` and `endNumber`)
  - `$Time$` → Segment timestamp (accumulates across `<S>` elements in SegmentTimeline)
- Handle **SegmentTimeline** with proper time persistence across `<S>` elements
- Respect `endNumber` attribute to limit segment generation
- Include initialization URLs when present (grouped with segment URLs as first item)
- Handle **BaseURL-only Representations**: When Representation contains only BaseURL (no SegmentList/SegmentTemplate), treat BaseURL as single segment URL

### Output Format
- JSON object mapping Representation IDs to arrays of resolved segment URLs
- Initialization URL (if exists) should be first item in the array
- Format: `{"representation_id": ["init_url", "segment1_url", "segment2_url", ...]}`
- For BaseURL-only representations: `{"representation_id": ["single_segment_url"]}`

### XML Structure Handling
- Properly handle nested `<Initialization sourceURL="..."/>` elements in SegmentList
- Support formatted number patterns like `$Number%05d$` in templates
- Handle template inheritance from AdaptationSet to Representation level

### Key Implementation Details
- Time values in SegmentTimeline must persist and accumulate across `<S>` elements
- When `t` attribute is present in `<S>` element, use it as starting time
- EndNumber takes priority over duration-based segment count calculations
- Template variables should be resolved at the appropriate hierarchy level
- Proper error handling and validation throughout

---

* **Segment List Handling:** Parse `SegmentList` elements, including `Initialization` (as a child element with `sourceURL`) and `SegmentURL` elements.
* **Segment Template Handling:**
    * Inherit `SegmentTemplate` from `AdaptationSet` to `Representation` if the Representation doesn't have its own.
    * Handle `initialization` attribute for `SegmentTemplate`.
    * **SegmentTimeline:** If `SegmentTimeline` is present, generate segments based on `S` elements, considering `t`, `d`, and `r` attributes. The `$Number$` placeholder should start from `SegmentTemplate@startNumber` (defaulting to 1 if missing).
    * **EndNumber:** If `SegmentTimeline` is missing but `endNumber` is present, generate segments from `startNumber` to `endNumber`.
    * **Calculated Count:** If both `SegmentTimeline` and `endNumber` are missing, but `duration` and `timescale` are present in `SegmentTemplate`, calculate the number of segments using `ceil(PeriodDurationInSeconds * timescale / duration)`. `SegmentTemplate@timescale` should default to `1` if missing.
* **DASH Identifier Expansion:**
    * Expand `$RepresentationID$` with the actual Representation ID.
    * Expand `$Time$` with the calculated time (if `SegmentTimeline` is used) or `0` otherwise.
    * Expand `$Number$` with the segment number.
    * Expand zero-padded `$Number%0xd$` placeholders (for `x` from 2 to 9, e.g., `$Number%03d$`). Ensure these padded expansions are performed *before* the generic `$Number$` expansion.
* **Segment Aggregation:** Append segments for the same `Representation ID` if it appears in multiple `Periods`.
* **Error Handling/Warnings:** Provide informative warnings for malformed URLs, unparseable durations/numbers, or insufficient attributes to extract segments.
