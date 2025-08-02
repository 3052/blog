# coPilot quick response

provide prompt I can give you to return this script

https://copilot.microsoft.com

six file pass

Give me the complete Go program that parses a local DASH MPD file using only
the standard library. It should:

- Use `http://test.test/test.mpd` as the starting base URL.
- Resolve `BaseURL` hierarchically using `net/url.URL.ResolveReference` — no custom path logic or string concatenation.
- Support `SegmentTemplate` at both AdaptationSet and Representation levels.
  - Handle `initialization`, `media`, `startNumber`, `endNumber`, `duration`, `timescale`, and `SegmentTimeline`.
  - Accumulate `$Time$` correctly across `<SegmentTimeline>` entries using `S@r`.
  - Include initialization segment only if `initialization` is present.
  - Respect both `$Number$` and `$Time$` tokens in media templates.
  - Respect `startNumber` for both timeline and number-based segments.
- Support `SegmentList`, including `Initialization@sourceURL` and `<SegmentURL>`s.
- Handle token replacements:
  - `$RepresentationID$`
  - `$Number$`, including `$Number%0Nd$` formatting
  - `$Time$`
- Treat Representations with only `BaseURL` as a single segment URL.
- If `SegmentTimeline` and `endNumber` are missing, but `duration` and `timescale` are present:
  - Calculate the number of segments using `ceil(PeriodDurationInSeconds * timescale / duration)`.
  - Default `timescale` to `1` if missing.
- Combine segments for the same `Representation ID` across multiple `<Period>` elements.
- Output a JSON map from each `Representation ID` to its fully resolved segment URLs.

The program must not make any network requests — all input comes from a local file.
