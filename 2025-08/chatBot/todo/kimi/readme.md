# kimi

provide markdown prompt I can give you in the future to return this script

https://kimi.com

six file pass

Return the **exact, complete Go source** for the program `dashmpd.go` that:

- Accepts a single command-line argument: the local path to an `.mpd` file.
- Starts every BaseURL resolution chain from the hard-coded root  
  `http://test.test/test.mpd`.
- Uses **only** `net/url.URL.ResolveReference` for every URL resolution.
- Supports `<SegmentTemplate>` and `<SegmentList>` on `<AdaptationSet>` **or** `<Representation>`.
- Expands all DASH identifiers:  
  `$RepresentationID$`, `$Number$`, `$Time$`, and zero-padded `$Number%0xd$` for `d = 1..9`.
- Generates the segment list as follows:
  - If `@endNumber` is present, use `@startNumber … @endNumber`.  
  - If `<SegmentTimeline>` is present, iterate each `<S>` exactly `1 + @r` times.  
  - Otherwise, use  
    `ceil(PeriodDurationInSeconds * timescale / duration)`  
    where **Period@duration is preferred** over MPD@mediaPresentationDuration.
- Emits the initialization URL (`@initialization` or `<Initialization sourceURL="">`) when present.
- Outputs **pure JSON** to stdout:  
  `{"RepresentationID":["init_url","seg1","seg2",…],…}` (single line, no HTML escaping).
- Never ignores any error—panic on failure is acceptable.
- If a `<Representation>` has neither template nor list, emit the absolute URL derived from its effective BaseURL.
- Distinguishes absent `startNumber` (default 1) from explicit `startNumber="0"`.
- Appends segments for the same Representation ID if it appears in multiple Periods.
- Respects `MPD@BaseURL`, `Period@BaseURL`, `AdaptationSet@BaseURL`, and `Representation@BaseURL`.
