# coPilot quick response

provide prompt I can give you to return this script

https://copilot.microsoft.com

one file pass

Please give me the complete Go program (standard library only) that:
- Reads an MPEG-DASH MPD XML file path from the CLI (`go run main.go <mpd_file_path>`)
- Starts with `const defaultBase = "http://test.test/test.mpd"`
- Resolves all nested `<BaseURL>` tags using only `net/url.Parse` and `net/url.URL.ResolveReference`
- Outputs a JSON map from each `Representation.ID` to its ordered list of fully resolved segment URLs
- Supports both `<SegmentList>` and `<SegmentTemplate>` elements from either AdaptationSet or Representation
- Handles `$RepresentationID$`, `$Number$`, `$Number%0Nd$`, `$Time$`, `$Time%0Nd$` placeholders
- Accumulates `$Time$` across `<SegmentTimeline><S>` elements
