# gemini flash

https://gemini.google.com

1. Go language script, input is path to local DASH MPD file
2. output is JSON object, key is Representation ID, value is segment URLs
3. standard library only
4. assume MPD URL is http://test.test/test.mpd
5. assume URLs contain path and query string
6. Representation with only BaseURL means one segment
7. Representations can be split across Periods
8. SegmentTemplate is a child of AdaptationSet or Representation
9. SegmentTemplate@timescale is 1 if missing
10. SegmentTemplate@startNumber is 1 if missing
11. $Number$ value should increase by 1 each iteration
12. SegmentTemplate@endNumber can exist. if so it defines the last segment
13. if no SegmentTemplate@endNumber use SegmentTimeline
13. if no SegmentTimeline use
   ceil(Period@duration * SegmentTemplate@timescale / SegmentTemplate@duration)
