# gemini flash

https://gemini.google.com

1. Go language script, input is path to local DASH MPD file
2. output is segment URLs as JSON, grouped by Representation
3. include logging to "log.txt"
4. assume MPD URL is "http://test.test/test.mpd" to resolve relative URLs
5. should work with attached files
6. Representation that contain only BaseURL should treat that as the only segment
7. SegmentTemplate.timescale is 1 if missing
8. consolidate Representations that are split across Periods
