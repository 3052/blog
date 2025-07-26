# deepSeek

provide prompt in markdown I can give you in the future to return this script

https://chat.deepseek.com

one file pass

Return the complete Go source for a DASH MPD segment expander that satisfies all these requirements:

### Input Requirements
1. Accepts path to local `.mpd` file as first CLI argument
2. Uses original MPD URL `http://test.test/test.mpd` as BaseURL starting point

### Output Requirements
1. Pure JSON to stdout: `{"RepresentationID":["absolute_url",...],...}`
2. All URLs must be absolute using proper `net/url.ResolveReference`

### MPD Processing Requirements
1. Handles both segment modes:
   - `<SegmentTimeline>` with precise timing
   - Simple `@startNumber`/`@duration` mode
2. Supports template expansions:
   - `$RepresentationID$`
   - `$Number$` and `$Number%0xd$` (padding)
   - `$Time$` and `$Time%0xd$` (padding)
3. Correctly increments `$Time$` across `<S>` elements in `<SegmentTimeline>`

### Implementation Requirements
1. Strict error handling for all operations:
   - XML parsing
   - URL resolution
   - Number conversions
2. Proper BaseURL resolution hierarchy
3. Clean ISO 8601 duration parsing (PT#H#M#S format)
4. Well-commented code with clear structure

### Code Style
1. No complex regex for duration parsing
2. Proper Go idioms and formatting
3. No external dependencies beyond stdlib

---

3. Resolves the BaseURL hierarchy (`MPD → Period → AdaptationSet → Representation`).  
4. Expands:  
   - `<SegmentList>` that may appear on `<AdaptationSet>` **or** `<Representation>`  
   - When neither `@endNumber` nor `<SegmentTimeline>` are present, derives the segment count as  
     `ceil(PeriodDurationInSeconds * SegmentTemplate@timescale / SegmentTemplate@duration)`  
     (use `SegmentTemplate@timescale = 1` if absent).  
   - Missing `@startNumber` (attribute absent) defaults to **1**; explicit `startNumber="0"` is honoured.  
5. If a `<Representation>` has **neither** `<SegmentTemplate>` **nor** `<SegmentList>`, emit the single absolute URL derived from its effective BaseURL.  
7. All other diagnostics (usage help, errors) go to stderr.  
8. **Eliminates duplicate segment URLs for each representation**, even when the same Representation ID appears in multiple Periods.  
9. **No external dependencies** beyond the Go standard library.  
10. **Appends** segments for the same Representation ID if it appears in multiple Periods.  
13. Distinguishes **absent** `startNumber` (default 1) from **explicit** `startNumber="0"`.  
14. `$Number$` **always** equals the segment number (starting from `startNumber`).  
15. `$Time$` **always** equals the presentation-time offset (in timescale units).  
16. Correctly replaces `$Number%0xd$` and `$Time%0xd$` with zero-padded values.  
18. Every `strconv`, `url.Parse`, `ParseFloat`, etc. **must** be checked and its error propagated.
