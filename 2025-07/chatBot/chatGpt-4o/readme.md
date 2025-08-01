# chatGpt

provide markdown prompt I can give you to return this script

https://chatgpt.com

two file pass

Please return the full Go script that:

- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)
- Resolves nested `<BaseURL>` elements using only `net/url.URL.ResolveReference`
- Handles `SegmentTemplate` substitution of `$RepresentationID$`, `$Number$`, `$Time$` and their formatted forms like `$Number%05d$`
- Supports `SegmentTimeline`, including `r` (repeat) and `t` (time offset) attributes
- Respects `startNumber` and `endNumber` to control how many segments are generated
- Falls back to generating 5 segments if no `SegmentTimeline` or `endNumber` is present
- Supports `<SegmentList>` and `<Initialization>` elements
- Falls back to using the `BaseURL` segment if no other segment info is present
- Uses only the Go standard library

## prompts

- Uses only the Go **standard library**
- Supports:
  - `<SegmentTemplate>` on both `AdaptationSet` and `Representation`, with inheritance
  - `<SegmentTimeline>` with proper handling of `@t`, `@d`, `@r`
  - `<BaseURL>` hierarchy: MPD → Period → AdaptationSet → Representation
  - Always uses `url.URL.ResolveReference` for URL resolution
  - Appends segments across multiple `<Period>`s for the same `Representation@id`
  - Defaults `timescale=1` if not present
  - If both `SegmentTimeline` and `endNumber` are missing, and both `duration`
    and `timescale` are present, calculates number of segments as:
   `ceil(PeriodDurationInSeconds * timescale / duration)`
