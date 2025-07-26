# kimi

provide prompt in markdown I can give you in the future to return this script

https://kimi.com

works with three files

Return the **complete, final Go source** for the DASH MPD segment expander that satisfies **every** requirement below.  
Do **not** ignore any errors (every `strconv.Atoi`, `url.Parse`, etc. **must** be checked and propagated).

1. Accepts the **path to a local `.mpd` file** as the **first CLI argument**.  
2. Starts the BaseURL resolution chain from the original MPD URL  
   `http://test.test/test.mpd `  
   **regardless** of any `BaseURL` attribute inside the document.  
3. Resolves the BaseURL hierarchy (`MPD → Period → AdaptationSet → Representation`).  
4. Expands:  
   - `<SegmentTemplate>` that may appear on `<AdaptationSet>` **or** `<Representation>`  
   - `<SegmentList>` that may appear on `<AdaptationSet>` **or** `<Representation>`  
   - Supports both `<SegmentTimeline>` and simple `@startNumber … @endNumber` modes.  
   - Expands `$RepresentationID$`, `$Number$`, `$Time$`, and `%0xd` padding.  
   - When neither `@endNumber` nor `<SegmentTimeline>` are present, derives the segment count as  
     `ceil(PeriodDurationInSeconds * SegmentTemplate@timescale / SegmentTemplate@duration)`  
     (use `SegmentTemplate@timescale = 1` if absent).  
   - Missing `@startNumber` (attribute absent) defaults to **1**; explicit `startNumber="0"` is honoured.  
5. If a `<Representation>` has **neither** `<SegmentTemplate>` **nor** `<SegmentList>`, emit the single absolute URL derived from its effective BaseURL.  
6. Outputs **pure JSON** to stdout:  
   `{"RepresentationID":["absolute_url",…],…}`  
7. All other diagnostics (usage help, errors) go to stderr.  
8. **Eliminates duplicate segment URLs for each representation**, even when the same Representation ID appears in multiple Periods.  
9. **No external dependencies** beyond the Go standard library.  
10. **Appends** segments for the same Representation ID if it appears in multiple Periods.  
11. Correctly **increments `$Time$`** across `<S>` elements in a `<SegmentTimeline>`.  
12. Handles full ISO-8601 durations (e.g., `PT2H13M19.040S`).  
13. Distinguishes **absent** `startNumber` (default 1) from **explicit** `startNumber="0"`.  
14. `$Number$` **always** equals the segment number (starting from `startNumber`).  
15. `$Time$` **always** equals the presentation-time offset (in timescale units).  
16. Correctly replaces `$Number%0xd$` and `$Time%0xd$` with zero-padded values.
