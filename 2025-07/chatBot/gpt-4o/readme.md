# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

six file pass

Please return the full Go script that:

- Uses only the Go **standard library**
- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)
- Supports:
  - `<SegmentTemplate>` on both `AdaptationSet` and `Representation`, with inheritance
  - `$RepresentationID$`, `$Number$`, `$Time$` substitutions, including formatted forms like `$Number%05d$`
  - `<SegmentTimeline>` with proper handling of `@t`, `@d`, `@r`
  - `<SegmentList>` and `<Initialization>` elements
  - `<BaseURL>` hierarchy: MPD → Period → AdaptationSet → Representation
  - Fallback to `BaseURL` segment if no other segment info is present
  - Always uses `url.URL.ResolveReference` for URL resolution
  - Appends segments across multiple `<Period>`s for the same `Representation@id`
  - Defaults `timescale=1` if not present
  - If both `SegmentTimeline` and `endNumber` are missing, and both `duration` and `timescale` are present, calculates number of segments as:
    ```
    ceil(PeriodDurationInSeconds * timescale / duration)
    ```
