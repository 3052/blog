# coPilot quick response

provide prompt I can give you to return this script

https://copilot.microsoft.com

one file pass

Please give me the complete Go program (using only the standard library) that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Resolves `BaseURL` hierarchically: MPD → Period → AdaptationSet → Representation
- Supports both `<SegmentList>` and `<SegmentTemplate>` (including `<SegmentTimeline>` and respecting `endNumber`)
- Handles all the following tokens: `$RepresentationID$`, `$Number$`, `$Number%0Nd$`, and `$Time$`
- Uses the starting base URL: `http://test.test/test.mpd`
- Outputs a **JSON map** from each `Representation.ID` to its ordered list of fully resolved segment URLs
