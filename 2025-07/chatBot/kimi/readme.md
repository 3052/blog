# kimi

provide markdown prompt I can give you in the future to return this script, the
prompt should assume you have no knowledge of this current chat

https://kimi.com

five file pass

Return the **exact Go source** for a command-line utility named `dashmpd` that:

1. Accepts the path to a local `.mpd` file as the **sole command-line argument**.
2. Starts every BaseURL resolution chain from the hard-coded root URL  
   `http://test.test/test.mpd`, regardless of any `<BaseURL>` element in the
   document.
3. Uses **only** `net/url.URL.ResolveReference` for every URL resolution step.
4. Supports both `<SegmentTemplate>` and `<SegmentList>` appearing on
   `<AdaptationSet>` **or** `<Representation>`.
5. Expands **all** DASH identifiers:
   - `$RepresentationID$`, `$Number$`, `$Time$`
   - and zero-padded `$Number%0xd$` for `d = 1..9`.
6. Derives the real segment list:
   - Use `<SegmentTimeline>` when present, iterating each `<S>` exactly `1 + @r`
     times.
   - If `@endNumber` is present, use `@startNumber … @endNumber`.
   - Otherwise use  
     `ceil(PeriodDurationInSeconds * timescale / duration)`  
     where **Period@duration is preferred** over MPD@mediaPresentationDuration.
7. Emits the initialization URL (`@initialization` or
   `<Initialization sourceURL="">`) when it exists.
8. Outputs **pure JSON** to stdout:  
   `{"RepresentationID":["init_url","seg1","seg2",…],…}`  
   (single line, no HTML escaping).
9. **Never ignores any error** — panic on failure is acceptable.
10. If a `<Representation>` has neither `<SegmentTemplate>` nor `<SegmentList>`,
    emit the single absolute URL derived from its effective BaseURL.
11. Distiguishes **absent** `startNumber` (default 1) from **explicit**
    `startNumber="0"`.
12. **Appends** segments for the same Representation ID if it appears in
    multiple Periods.

---

4. Expands:  
   - `<SegmentList>` that may appear on `<AdaptationSet>` **or** `<Representation>`  
   - When neither `@endNumber` nor `<SegmentTimeline>` are present, derives the segment count as  
     `ceil(PeriodDurationInSeconds * SegmentTemplate@timescale / SegmentTemplate@duration)`  
     (use `SegmentTemplate@timescale = 1` if absent).  
   - Missing `@startNumber` (attribute absent) defaults to **1**; explicit `startNumber="0"` is honoured.  
7. All other diagnostics (usage help, errors) go to stderr.  
9. **No external dependencies** beyond the Go standard library.  
12. Handles full ISO-8601 durations (e.g., `PT2H13M19.040S`).  
14. `$Number$` **always** equals the segment number (starting from `startNumber`).  
15. `$Time$` **always** equals the presentation-time offset (in timescale units).  
16. Correctly replaces `$Number%0xd$` and `$Time%0xd$` with zero-padded values.  
18. Every `strconv`, `url.Parse`, `ParseFloat`, etc. **must** be checked and its error propagated.
