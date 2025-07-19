# gemini flash

https://gemini.google.com

1. Go language script, input is path to local DASH MPD file
2. output is segment URLs as JSON, grouped by Representation
3. assume MPD URL is "http://test.test/test.mpd" to resolve relative URLs
4. Representation that contain only BaseURL should treat that as the only segment
5. SegmentTemplate is a child of AdaptationSet or Representation
6. SegmentTemplate.timescale is 1 if missing
7. consolidate Representations that are split across Periods
8. when replacing $Number$ the value should increase by 1 each time
