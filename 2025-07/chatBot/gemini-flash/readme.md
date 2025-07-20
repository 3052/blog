# gemini flash

https://gemini.google.com

1. Go language script, input is path to local DASH MPD file
2. output is segment URLs as JSON, grouped by Representation
3. use MPD URL "http://test.test/test.mpd" to resolve relative URLs
4. Representation with only BaseURL means one segment
5. Representations can be split across Periods
6. SegmentTemplate is a child of AdaptationSet or Representation
7. SegmentTemplate@timescale is 1 if missing
8. SegmentTemplate@endNumber defines the last segment when it exists
9. if no SegmentTemplate@endNumber use SegmentTimeline
10. if no SegmentTimeline use
   ceil(Period@duration * SegmentTemplate@timescale / SegmentTemplate@duration)
