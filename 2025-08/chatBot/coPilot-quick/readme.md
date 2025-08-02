# coPilot quick response

provide prompt I can give you to return this script

https://copilot.microsoft.com

two file pass

Give me the complete Go program that parses a DASH MPD using only the standard
library, uses `http://test.test/test.mpd` as the starting base URL, resolves
BaseURL hierarchically using `net/url.URL.ResolveReference`, supports
SegmentTemplate (at both AdaptationSet and Representation levels) including
SegmentTimeline, Initialization, `startNumber`, and `endNumber`, supports
SegmentList including Initialization@sourceURL, handles token replacements
(`$RepresentationID$`, `$Number$`, `$Number%0Nd$`, `$Time$`), accumulates
`$Time$` correctly across `<SegmentTimeline>` entries by using each `<S>`
element `1 + S@r` times, treats BaseURL-only Representations as single segment
URLs, and outputs a JSON map from each Representation ID to its fully resolved
segment URLs.
