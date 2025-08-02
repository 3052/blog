# chatGpt

provide prompt I can give you to return this script

https://chatgpt.com

one file pass

Please generate a complete Go program (main.go) using only the standard library
that:

1. Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
2. Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
3. Chains BaseURL elements through MPD → Period → AdaptationSet → Representation
4. Supports `<SegmentList>` and `<SegmentTemplate>` at both AdaptationSet and Representation levels
5. Handles `media`, `initialization`, `startNumber`, `endNumber`, and `<SegmentTimeline>` (with `t`, `d`, `r`) attributes
6. Substitutes `$RepresentationID$`, `$Number$`, and `$Time$` into template URLs
7. Prepends any initialization segment URL (with `$RepresentationID$` replaced)
8. Resolves every segment URL to an absolute URL and collects them in order
9. Prints a JSON object mapping each Representation.ID to its ordered slice of fully-resolved segment URLs
10. Includes full error handling (exit nonzero on any parse/URL error, logging to stderr)
