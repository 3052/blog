# gemini flash

https://gemini.google.com

1. Go language script, input is path to local DASH MPD file
2. output is segment URLs as JSON, grouped by Representation
3. use MPD URL "http://test.test/test.mpd" to resolve relative URLs
4. $Number$ in SegmentTemplate.media should increase by 1 each time
5. Representation with only BaseURL means one segment
6. Representations can be split across Periods
7. SegmentTemplate is a child of AdaptationSet or Representation
8. SegmentTemplate.endNumber defines the last segment when it exists
9. SegmentTemplate.timescale is 1 if missing
10. SegmentTimeline defines the segments when it exists
