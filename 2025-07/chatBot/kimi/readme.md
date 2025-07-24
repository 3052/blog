# kimi

https://kimi.com

Return the **single-file Go program** that:

1. Accepts the path to a local `.mpd` file as its **first CLI argument**.
2. Starts from the original MPD URL `http://test.test/test.mpd `.
3. Resolves the **BaseURL hierarchy** (`MPD → Period → AdaptationSet → Representation`) **exclusively** with `net/url.Parse` + `ResolveReference`.
4. Expands any `<SegmentTemplate>` that may appear on `<AdaptationSet>` **or** `<Representation>`:
   - Supports both `<SegmentTimeline>` and simple `@startNumber … @endNumber` modes.  
   - Expands `$RepresentationID$`, `$Number$`, `$Time$`, and `%0xd` padding.  
   - **When neither `@endNumber` nor `<SegmentTimeline>` are present**, derive the last segment number as  
     `ceil(PeriodDurationInSeconds * SegmentTemplate@timescale / SegmentTemplate@duration)`  
     (use `SegmentTemplate@timescale = 1` if absent).  
   - Missing `@startNumber` (attribute absent) is **1**; explicit `startNumber="0"` is honoured.
5. If a `<Representation>` has no `<SegmentTemplate>` anywhere in its hierarchy, emit the single absolute URL derived from its effective BaseURL.
6. Outputs **pure JSON** to stdout:  
   `{"RepresentationID":["absolute_url",…],…}`
7. All other diagnostics (usage help, errors) go to stderr.
8. Eliminates duplicate segment URLs for each representation.

provide prompt in markdown I can give you in the future to return this script
