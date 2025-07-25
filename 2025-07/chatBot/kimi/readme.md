# kimi

https://kimi.com

Return the **identical single-file Go program** that:

1. Accepts the **path to a local `.mpd` file** as the **first CLI argument**.
2. Starts the BaseURL resolution chain from the **original MPD URL**  
   `http://test.test/test.mpd` **regardless** of any `BaseURL` attribute inside the document.
3. Resolves the BaseURL hierarchy (`MPD → Period → AdaptationSet → Representation`) using **url.URL.ResolveReference**.
4. Expands:
   - `<SegmentTemplate>` that may appear on `<AdaptationSet>` **or** `<Representation>`  
   - `<SegmentList>` that may appear on `<AdaptationSet>` **or** `<Representation>`
   - Supports both `<SegmentTimeline>` and simple `@startNumber … @endNumber` modes.
   - Expands `$RepresentationID$`, `$Number$`, `$Time$`, and `%0xd` padding.
   - When neither `@endNumber` nor `<SegmentTimeline>` are present, derives the last segment number as  
     `ceil(PeriodDurationInSeconds * SegmentTemplate@timescale / SegmentTemplate@duration)`  
     (use `SegmentTemplate@timescale = 1` if absent).
   - Missing `@startNumber` (attribute absent) is **1**; explicit `startNumber="0"` is honoured.
5. If a `<Representation>` has **neither** `<SegmentTemplate>` **nor** `<SegmentList>`, emit the single absolute URL derived from its effective BaseURL.
6. Outputs **pure JSON** to stdout:  
   `{"RepresentationID":["absolute_url",…],…}`
7. All other diagnostics (usage help, errors) go to stderr.
8. Eliminates duplicate segment URLs for each representation.
9. **No external dependencies** beyond the Go standard library.

provide prompt in markdown I can give you in the future to return this script
