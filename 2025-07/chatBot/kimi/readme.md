# kimi

provide prompt in markdown I can give you in the future to return this script

https://kimi.com

two file pass

Return the **exact** Go source for the DASH MPD segment expander that satisfies **all** of the following requirements **without** adding any extra explanation:

1. Accepts the **path to a local `.mpd` file** as the **first CLI argument**.  
2. Starts the BaseURL resolution chain from the fixed URL  
   `http://test.test/test.mpd ` regardless of any `<BaseURL>` attribute inside the document.  
3. Uses **only** `net/url.URL.ResolveReference` for every URL resolution step.  
4. Supports `<SegmentTemplate>` on **either** `<AdaptationSet>` or `<Representation>`.  
5. Supports `<SegmentList>` on **either** `<AdaptationSet>` or `<Representation>`.  
6. Expands all DASH identifiers:  
   `$RepresentationID$`, `$Number$`, `$Time$`, and zero-padded `%0xd`.  
7. Calculates the **real segment list**:  
   • use `<SegmentTimeline>` when present, iterating **exactly** `1 + @r` times for each `<S>`;  
   • otherwise use `@duration/@timescale` with the duration derived from **MPD@mediaPresentationDuration** or **Period@duration** (ISO-8601);  
   • if `@endNumber` is present, use `@startNumber … @endNumber` **instead** of duration math.  
8. Emits the initialization URL (`@initialization` or `<Initialization>@sourceURL`) when it exists.  
9. Outputs **pure JSON** to stdout:  
   `{"RepresentationID":["init_url","seg1","seg2",…],…}`  
   (single line, no HTML escaping).  
10. **Never ignore any error** — every `strconv`, `url.Parse`, etc. must be checked and propagated (panic on failure is acceptable).  
11. If a `<Representation>` has **neither** `<SegmentTemplate>` **nor** `<SegmentList>`, emit the single absolute URL derived from its effective BaseURL.

---

4. Expands:  
   - `<SegmentList>` that may appear on `<AdaptationSet>` **or** `<Representation>`  
   - When neither `@endNumber` nor `<SegmentTimeline>` are present, derives the segment count as  
     `ceil(PeriodDurationInSeconds * SegmentTemplate@timescale / SegmentTemplate@duration)`  
     (use `SegmentTemplate@timescale = 1` if absent).  
   - Missing `@startNumber` (attribute absent) defaults to **1**; explicit `startNumber="0"` is honoured.  
7. All other diagnostics (usage help, errors) go to stderr.  
9. **No external dependencies** beyond the Go standard library.  
10. **Appends** segments for the same Representation ID if it appears in multiple Periods.  
12. Handles full ISO-8601 durations (e.g., `PT2H13M19.040S`).  
13. Distinguishes **absent** `startNumber` (default 1) from **explicit** `startNumber="0"`.  
14. `$Number$` **always** equals the segment number (starting from `startNumber`).  
15. `$Time$` **always** equals the presentation-time offset (in timescale units).  
16. Correctly replaces `$Number%0xd$` and `$Time%0xd$` with zero-padded values.  
18. Every `strconv`, `url.Parse`, `ParseFloat`, etc. **must** be checked and its error propagated.
